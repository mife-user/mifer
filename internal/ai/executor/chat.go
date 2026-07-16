package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"mifer/internal/ai/agent"
	aicallback "mifer/internal/ai/callback"
	"mifer/internal/ai/confirm"
	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// Chat 执行一次对话，通过 callback 将事件实时传递到上层。
// 根据 req.Mode 路由："" 为普通对话，"plan" 为先制定计划等确认再执行。
func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	logger.Debug("Chat", logger.S("mode", req.Mode), logger.S("content_preview", req.Content[:min(len(req.Content), 50)]))

	// 检查模型是否可用（api_key 未配置时 Runner 为 nil）
	if e.Runner == nil {
		_ = callback("response", "当前apikey未配置，AI对话功能暂不可用。请使用 /config 命令编辑配置文件，在 ai.backends.default.api_key 填入您的API Key后重载配置。")
		return nil
	}

	ctx, sessionID, toolCB, err := e.prepareChat(c, req, callback)
	if err != nil {
		return err
	}
	defer e.Humen.ConfirmStore.Cleanup(sessionID)

	// plan 模式路由
	if req.Mode == "plan" {
		return e.runPlanFlow(ctx, req, toolCB, callback, sessionID)
	}

	// 普通对话模式

	result, err := e.runConversation(ctx, req, toolCB, callback)
	if err != nil {
		logger.Info("chat", logger.C(err))
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	logger.Debug("chat", logger.I("eventCount", result.eventCount))

	if result.lastMsg == "" {
		logger.Error("chat", logger.S("lastmsg", result.lastMsg))
		return errorer.New(errorer.ErrCallBackNull)
	}

	e.finalizeChat(result.lastMsg, req.Content)
	return nil
}

// prepareChat 对话前置处理：压缩检查、session 切换、追加用户消息、构建 tool callback 和 context。
func (e *Executor) prepareChat(
	c context.Context,
	req *domain.TalkReq,
	callback func(event, content string) error,
) (context.Context, string, callbacks.Handler, error) {
	// 检查是否需要压缩上下文（上一轮结束时已标记）
	if e.compress.needsCompression {
		if err := e.compress.compressor.Compress(
			c, e.Humen.Prompt.Memory, e.Humen.Prompt.SystemPrompt,
			e.compress.lastPromptTokens, callback,
		); err != nil {
			logger.Error("compress", logger.C(err))
		}
		e.compress.needsCompression = false
	}

	// 指定了 sessionID 时先切换记忆，保证 switch + chat 原子执行
	if req.SessionID != "" {
		if err := e.Humen.Prompt.Memory.SwitchSession(req.SessionID); err != nil {
			logger.Error("memory", logger.S("session", req.SessionID), logger.C(err))
		}
	}

	e.Humen.Prompt.Memory.AppendUser(req.Content)

	// 获取会话 ID 用于工具确认和清理
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID, _ = c.Value("id").(string)
	}

	// 构建 per-invocation Tool 回调处理器
	toolCB := aicallback.NewHandler(callback)

	// 将 executor 回调注入 confirm 中间件的 context key
	ctx := confirm.WithCallback(c, confirm.ExecutorCallback(callback))
	ctx = confirm.WithSessionID(ctx, sessionID)

	return ctx, sessionID, toolCB, nil
}

// finalizeChat 对话收尾：追加 assistant 回复、持久化记忆、自动重命名、保存快照、触发习惯总结。
func (e *Executor) finalizeChat(lastMsg, userMsg string) {
	e.Humen.Prompt.Memory.AppendAssistant(lastMsg)
	if err := e.Humen.Prompt.Memory.Save(); err != nil {
		logger.Error("memory", logger.C(err))
	}

	// 首次对话结束后自动用首条用户消息前缀重命名会话
	if e.Humen.Prompt.Memory.FileCreated() {
		if err := e.Humen.Prompt.Memory.AutoRenameFromFirstMessage(); err != nil {
			logger.Warn("memory", logger.C(err))
		}
	}

	// 轮次结束后保存文件快照（需在配置中启用 snapshot_enabled）
	if e.Snapshot != nil {
		round := e.Humen.Prompt.Memory.CountUserMessages()
		logger.Debug("snapshot", logger.I("round", round))
		if err := e.Snapshot.SaveRound(round); err != nil {
			logger.Warn("snapshot", logger.C(err), logger.I("round", round))
		} else {
			logger.Debug("snapshot", logger.I("round", round))
		}
	}

	// 异步总结用户习惯（fire-and-forget，不阻塞对话响应）
	if e.Humen.Graphs.Habit != nil {
		go e.summarizeHabits(userMsg, lastMsg)
	}
}

// summarizeHabits 异步调用习惯总结图，分析本轮对话并更新用户级 MIFER.md。
func (e *Executor) summarizeHabits(userMsg, assistantReply string) {
	ctx := context.Background()

	miferPath := filepath.Join(conf.GetConfig().Path.CfgPath, "MIFER.md")
	existingContent := ""
	if data, err := os.ReadFile(miferPath); err == nil {
		existingContent = string(data)
	}

	msgs := []*schema.Message{
		schema.SystemMessage(agent.HabitInstruction),
		schema.UserMessage(fmt.Sprintf(
			"## 本轮对话\n用户: %s\n\nAI: %s\n\n## 现有用户画像\n%s",
			userMsg, assistantReply, existingContent,
		)),
	}

	if _, err := e.Humen.Graphs.Habit.Invoke(ctx, msgs); err != nil {
		logger.Warn("用户习惯总结失败", logger.C(err))
	}
}

// checkCompressionThreshold 检查当前 PromptTokens 是否超过压缩阈值。
func (e *Executor) checkCompressionThreshold() {
	if e.compress.needsCompression {
		return
	}
	ctxCfg := conf.GetConfig().Ai.Context
	if ctxCfg.Length <= 0 {
		return
	}
	thresholdTokens := int(float64(ctxCfg.Length) * ctxCfg.Threshold)
	if e.Token.Prompt >= thresholdTokens {
		e.compress.needsCompression = true
		e.compress.lastPromptTokens = e.Token.Prompt
		logger.Warn("上下文超过压缩阈值",
			logger.I("prompt_tokens", e.Token.Prompt),
			logger.I("threshold", thresholdTokens),
			logger.I("limit", ctxCfg.Length),
		)
	}
}
