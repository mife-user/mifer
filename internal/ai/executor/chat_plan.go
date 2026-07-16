package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"mifer/internal/ai/agent"
	"mifer/internal/ai/confirm"
	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// runPlanFlow plan 模式的完整流程：制定计划 → 确认 → 执行。
// 两轮 memory save：plan 确认后保存一次（计划作为 assistant 消息），execute 后保存一次。
func (e *Executor) runPlanFlow(
	ctx context.Context,
	req *domain.TalkReq,
	toolCB callbacks.Handler,
	callback func(event, content string) error,
	sessionID string,
) error {
	// 1. 构建 PlanGraph 输入消息（系统提示 + 完整历史 + 任务）
	planMsgs := e.buildPlanMessages(req.Content)

	// 2. 注入 callback + sessionID → plan_confirm Lambda 通过 ctx 获取
	planCtx := confirm.WithCallback(ctx, confirm.ExecutorCallback(callback))
	planCtx = confirm.WithSessionID(planCtx, sessionID)

	// 告知 TUI 正在制定计划
	_ = callback("system", "正在分析项目并制定计划...")

	// 3. 执行 PlanGraph（阻塞等用户确认）
	planContent, err := e.Humen.Graphs.Plan.Invoke(planCtx, planMsgs)
	if err != nil {
		if errors.Is(err, agent.ErrPlanRejected) {
			_ = callback("system", "计划已取消")
			return nil
		}
		logger.Error("计划制定失败", logger.C(err))
		_ = callback("system", "计划制定失败: "+err.Error())
		return nil
	}

	// ★ 第一轮 save：计划制定完成 → 追加 assistant + 持久化
	planMsg := fmt.Sprintf("已制定计划并写入文件。\n\n%s", planContent)
	e.Humen.Prompt.Memory.AppendAssistant(planMsg)
	if err := e.Humen.Prompt.Memory.Save(); err != nil {
		logger.Error("保存计划记忆失败", logger.C(err))
	}

	// 4. 执行阶段：复用 runConversation（内存已含 plan 的 assistant 消息）
	result, err := e.runConversation(ctx, req, toolCB, callback)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	logger.Debug("plan 执行完成", logger.I("eventCount", result.eventCount), logger.I("msgLen", len(result.lastMsg)))

	if result.lastMsg == "" {
		return nil
	}

	// ★ 第二轮 save：execute 完成 → finalize
	e.finalizeChat(result.lastMsg, req.Content)
	return nil
}

// buildPlanMessages 构建 plan 阶段的输入消息（系统提示词 + 完整对话历史）。
func (e *Executor) buildPlanMessages(task string) []*schema.Message {
	planSystemPrompt := fmt.Sprintf(agent.PlanInstruction+"\n\n## 当前任务\n%s", task)

	msgs := make([]*schema.Message, len(e.Humen.Prompt.Memory.Messages())+1)
	msgs = append(msgs, schema.SystemMessage(planSystemPrompt))
	for _, m := range e.Humen.Prompt.Memory.Messages() {
		msgs = append(msgs, m)
	}
	return msgs
}

// readPlanFile 从 .mifer/plans/ 目录读取指定计划文件的内容。
func readPlanFile(name string) (string, error) {
	plansDir := filepath.Join(conf.GetConfig().Path.Workdir, ".mifer", "plans")
	data, err := os.ReadFile(filepath.Join(plansDir, name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
