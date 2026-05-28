package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"regexp"
	"sync"
	"time"
)

// ConfirmAction 确认动作类型
type ConfirmAction string

const (
	ActionAccept     ConfirmAction = "accept"     // 接受：仅本次放行
	ActionRefuse     ConfirmAction = "refuse"     // 拒绝：本次拒绝执行
	ActionAllow      ConfirmAction = "allow"      // 始终允许：执行并加入会话白名单
	ActionSupplement ConfirmAction = "supplement" // 补充：带修改意见重新生成
)

// ConfirmResult 确认结果
type ConfirmResult struct {
	Action ConfirmAction
}

// PlanConfirmResult 计划确认结果
type PlanConfirmResult struct {
	Action ConfirmAction
	Input  string // supplement 时携带的补充文本
}

// AllowToolRule 工具参数级别白名单规则（供外部使用）
type AllowToolRule struct {
	Tool        string
	ArgsPattern map[string]string
}

// compiledToolRule 编译后的工具白名单规则
type compiledToolRule struct {
	tool        string
	argsPattern map[string]*regexp.Regexp
}

// ConfirmBus 工具调用确认总线
//
// 管理待确认的工具调用（callID → 等待通道）和会话级动态白名单。
// 工具中间件调用 Confirm() 阻塞等待 TUI 响应，HTTP handler 调用 Resolve() 解除阻塞。
type ConfirmBus struct {
	mu          sync.Mutex
	sseMu       sync.Mutex                         // 串行化 sendSSE 调用，避免并发写入 HTTP ResponseWriter
	pending     map[string]chan ConfirmResult     // callID → 工具确认等待通道
	planPending map[string]chan PlanConfirmResult // callID → 计划确认等待通道
	allowlist   map[string]bool                   // 会话级动态白名单（运行时通过 allow 动作添加）
	toolRules   []compiledToolRule                // 参数级别白名单规则（只读，构造时编译）
	sendSSE     func(event, data string) error     // SSE 回调（每次 Chat 调用前设置）
}

// NewConfirmBus 创建确认总线，persistAllowlist 为配置中的持久白名单，rules 为参数级白名单规则
func NewConfirmBus(persistAllowlist []string, rules []AllowToolRule) *ConfirmBus {
	cb := &ConfirmBus{
		pending:     make(map[string]chan ConfirmResult),
		planPending: make(map[string]chan PlanConfirmResult),
		allowlist:   make(map[string]bool),
	}
	for _, name := range persistAllowlist {
		cb.allowlist[name] = true
	}
	for _, rule := range rules {
		if rule.Tool == "" {
			continue
		}
		tr := compiledToolRule{
			tool:        rule.Tool,
			argsPattern: make(map[string]*regexp.Regexp),
		}
		for field, pattern := range rule.ArgsPattern {
			re, err := regexp.Compile(pattern)
			if err != nil {
				logger.Warn("白名单规则正则编译失败",
					logger.S("tool", rule.Tool),
					logger.S("field", field),
					logger.S("pattern", pattern),
					logger.C(err))
				continue
			}
			tr.argsPattern[field] = re
		}
		if len(tr.argsPattern) > 0 {
			cb.toolRules = append(cb.toolRules, tr)
		}
	}
	return cb
}

// SetSSECallback 设置 SSE 事件发送回调，每次 Chat 调用前由 executor 设置
func (cb *ConfirmBus) SetSSECallback(fn func(event, data string) error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.sendSSE = fn
}

// Confirm 确认工具调用，不在白名单中的工具会阻塞等待 TUI 响应
func (cb *ConfirmBus) Confirm(ctx context.Context, toolName, argsJSON string) error {
	// 检查白名单 → 直接放行
	cb.mu.Lock()
	if cb.allowlist[toolName] {
		cb.mu.Unlock()
		return nil
	}
	sendSSE := cb.sendSSE
	cb.mu.Unlock()

	// 检查结构化参数白名单 → 匹配则直接放行（无需确认）
	if cb.matchToolRules(toolName, argsJSON) {
		return nil
	}

	// 无 SSE 回调（非 TUI 模式）→ 直接放行
	if sendSSE == nil {
		return nil
	}

	// 生成唯一 callID
	callID := generateCallID()

	// 创建等待通道
	ch := make(chan ConfirmResult, 1)
	cb.mu.Lock()
	cb.pending[callID] = ch
	cb.mu.Unlock()

	// 发送 tool_confirm SSE 事件（串行化发送，避免并发写入 HTTP ResponseWriter）
	sendArgs := truncateArgs(argsJSON)
	payload := fmt.Sprintf("%s\x00%s\x00%s", callID, toolName, sendArgs)
	cb.sseMu.Lock()
	err := sendSSE("tool_confirm", payload)
	cb.sseMu.Unlock()
	if err != nil {
		cb.mu.Lock()
		delete(cb.pending, callID)
		cb.mu.Unlock()
		return err
	}

	logger.Debug("工具确认等待中",
		logger.S("callID", callID),
		logger.S("toolName", toolName))

	// 获取超时配置
	timeout := conf.GetConfig().Cli.Tui.ConfirmTimeout
	if timeout <= 0 {
		timeout = 60
	}
	timer := time.After(time.Duration(timeout) * time.Second)

	// 阻塞等待 TUI 响应或超时
	// 不使用 ctx.Done()：HTTP 请求上下文在 SSE 流式传输期间可能被底层取消，
	// 确认生命周期由 ConfirmTimeout 和 CancelAll() 控制
	var result ConfirmResult
	select {
	case result = <-ch:
		// 正常收到 TUI 响应
	case <-timer:
		cb.mu.Lock()
		delete(cb.pending, callID)
		cb.mu.Unlock()
		logger.Warn("工具确认超时，自动拒绝",
			logger.S("callID", callID),
			logger.S("toolName", toolName))
		return errors.New("工具确认超时，自动拒绝")
	}

	switch result.Action {
	case ActionAccept:
		return nil
	case ActionAllow:
		// 加入会话白名单
		cb.mu.Lock()
		cb.allowlist[toolName] = true
		cb.mu.Unlock()
		return nil
	case ActionRefuse:
		return errors.New("工具调用被用户拒绝")
	default:
		return errors.New("未知的确认动作: " + string(result.Action))
	}
}

// Resolve 由 HTTP handler 调用，解除 Confirm() 的阻塞
//
// 返回 false 表示 callID 无效或已被处理
func (cb *ConfirmBus) Resolve(callID string, action ConfirmAction) bool {
	cb.mu.Lock()
	ch, ok := cb.pending[callID]
	if ok {
		delete(cb.pending, callID)
	}
	cb.mu.Unlock()

	if !ok {
		logger.Warn("确认回调 callID 已过期",
			logger.S("callID", callID),
			logger.S("action", string(action)))
		return false
	}

	ch <- ConfirmResult{Action: action}
	logger.Info("工具确认已解决",
		logger.S("callID", callID),
		logger.S("action", string(action)))
	return true
}

// ConfirmPlan 向 TUI 发送计划确认请求并阻塞等待用户响应
//
// 与 Confirm 不同，ConfirmPlan 专门用于计划确认场景：
//   - SSE 事件为 plan_confirm，格式 callID\x00planContent
//   - 支持 accept / refuse / supplement 三种动作
//   - supplement 时通过 PlanConfirmResult.Input 携带补充文本
func (cb *ConfirmBus) ConfirmPlan(ctx context.Context, planContent string) (PlanConfirmResult, error) {
	sendSSE := cb.sendSSE
	if sendSSE == nil {
		return PlanConfirmResult{Action: ActionAccept}, nil
	}

	callID := "plan_" + generateCallID()
	ch := make(chan PlanConfirmResult, 1)
	cb.mu.Lock()
	cb.planPending[callID] = ch
	cb.mu.Unlock()

	// 发送 SSE: plan_confirm（串行化发送）
	payload := fmt.Sprintf("%s\x00%s", callID, planContent)
	cb.sseMu.Lock()
	err := sendSSE("plan_confirm", payload)
	cb.sseMu.Unlock()
	if err != nil {
		cb.mu.Lock()
		delete(cb.planPending, callID)
		cb.mu.Unlock()
		return PlanConfirmResult{}, err
	}

	logger.Debug("计划确认等待中", logger.S("callID", callID))

	// 阻塞等待 TUI 响应
	// 不使用 ctx.Done()：HTTP 请求上下文在 SSE 流式传输期间可能被底层取消
	result := <-ch
	return result, nil
}

// ResolvePlan 由 HTTP handler 调用，解除 ConfirmPlan 的阻塞
func (cb *ConfirmBus) ResolvePlan(callID string, action ConfirmAction, input string) bool {
	cb.mu.Lock()
	ch, ok := cb.planPending[callID]
	if ok {
		delete(cb.planPending, callID)
	}
	cb.mu.Unlock()

	if !ok {
		logger.Warn("计划确认回调 callID 已过期",
			logger.S("callID", callID),
			logger.S("action", string(action)))
		return false
	}

	ch <- PlanConfirmResult{Action: action, Input: input}
	logger.Info("计划确认已解决",
		logger.S("callID", callID),
		logger.S("action", string(action)))
	return true
}

// IsAllowed 检查工具名是否在白名单中
func (cb *ConfirmBus) IsAllowed(toolName string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.allowlist[toolName]
}

// matchToolRules 检查工具调用是否匹配参数级别白名单规则
//
// 遍历已编译的规则，若某规则的 tool 名称匹配且所有 argsPattern 正则均匹配对应参数字段值，则返回 true。
// 规则在 ConfirmBus 构造时编译，此后只读，无需加锁。
func (cb *ConfirmBus) matchToolRules(toolName, argsJSON string) bool {
	if len(cb.toolRules) == 0 || argsJSON == "" {
		return false
	}
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &argsMap); err != nil {
		return false
	}
	for _, rule := range cb.toolRules {
		if rule.tool != toolName {
			continue
		}
		allMatch := true
		for field, re := range rule.argsPattern {
			val, ok := argsMap[field]
			if !ok {
				allMatch = false
				break
			}
			strVal, ok := val.(string)
			if !ok {
				allMatch = false
				break
			}
			if !re.MatchString(strVal) {
				allMatch = false
				break
			}
		}
		if allMatch {
			logger.Debug("工具调用匹配参数白名单规则",
				logger.S("toolName", toolName),
				logger.S("argsJSON", argsJSON))
			return true
		}
	}
	return false
}

func generateCallID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// truncateArgs 截断过长的 argsJSON，避免 SSE 数据行超出客户端 scanner 缓冲区
func truncateArgs(argsJSON string) string {
	maxLen := conf.GetConfig().Cli.Tui.ConfirmPayloadMaxLen
	if maxLen <= 0 {
		maxLen = 2048
	}
	if len(argsJSON) <= maxLen {
		return argsJSON
	}
	return argsJSON[:maxLen] + "...(截断)"
}

// CancelAll 取消所有待处理的确认（工具确认 + 计划确认）
//
// 在 executor 异常退出时调用，解除阻塞的 Confirm / ConfirmPlan 调用，
// 避免 goroutine 泄漏和级联错误。
func (cb *ConfirmBus) CancelAll(reason string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for callID, ch := range cb.pending {
		delete(cb.pending, callID)
		select {
		case ch <- ConfirmResult{Action: ActionRefuse}:
		default:
		}
	}
	for callID, ch := range cb.planPending {
		delete(cb.planPending, callID)
		select {
		case ch <- PlanConfirmResult{Action: ActionRefuse}:
		default:
		}
	}
}
