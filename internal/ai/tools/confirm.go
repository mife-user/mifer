package tools

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

// ConfirmAction 确认动作类型
type ConfirmAction string

const (
	ActionAccept ConfirmAction = "accept" // 接受：仅本次放行
	ActionRefuse ConfirmAction = "refuse" // 拒绝：本次拒绝执行
	ActionAllow  ConfirmAction = "allow"  // 始终允许：执行并加入会话白名单
)

// ConfirmResult 确认结果
type ConfirmResult struct {
	Action ConfirmAction
}

// ConfirmBus 工具调用确认总线
//
// 管理待确认的工具调用（callID → 等待通道）和会话级动态白名单。
// 工具中间件调用 Confirm() 阻塞等待 TUI 响应，HTTP handler 调用 Resolve() 解除阻塞。
type ConfirmBus struct {
	mu        sync.Mutex
	pending   map[string]chan ConfirmResult // callID → 等待通道
	allowlist map[string]bool               // 会话级动态白名单（运行时通过 allow 动作添加）
	sendSSE   func(event, data string) error // SSE 回调（每次 Chat 调用前设置）
}

// NewConfirmBus 创建确认总线，persistAllowlist 为配置中的持久白名单
func NewConfirmBus(persistAllowlist []string) *ConfirmBus {
	cb := &ConfirmBus{
		pending:   make(map[string]chan ConfirmResult),
		allowlist: make(map[string]bool),
	}
	for _, name := range persistAllowlist {
		cb.allowlist[name] = true
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
func (cb *ConfirmBus) Confirm(toolName, argsJSON string) error {
	// 检查白名单 → 直接放行
	cb.mu.Lock()
	if cb.allowlist[toolName] {
		cb.mu.Unlock()
		return nil
	}
	sendSSE := cb.sendSSE
	cb.mu.Unlock()

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

	// 发送 tool_confirm SSE 事件
	payload := fmt.Sprintf("%s\x00%s\x00%s", callID, toolName, argsJSON)
	if err := sendSSE("tool_confirm", payload); err != nil {
		cb.mu.Lock()
		delete(cb.pending, callID)
		cb.mu.Unlock()
		return err
	}

	// 阻塞等待 TUI 响应
	result := <-ch

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
		return false
	}

	ch <- ConfirmResult{Action: action}
	return true
}

// IsAllowed 检查工具名是否在白名单中
func (cb *ConfirmBus) IsAllowed(toolName string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.allowlist[toolName]
}

func generateCallID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
