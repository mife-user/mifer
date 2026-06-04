package compressor

import (
	"context"
	"fmt"
	"strings"

	"mifer/internal/ai/memory"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Compress 执行上下文压缩：发送通知 → 调用压缩模型生成摘要 → 替换记忆 → 持久化
//
// 降级策略：技能缺失、模型不可用或调用失败时，移除最早的1轮对话
func (c *Compressor) Compress(
	ctx context.Context,
	mem *memory.Memory,
	systemPrompt string,
	lastPromptTokens int,
	callback func(event, content string) error,
) error {
	ctxCfg := conf.GetConfig().Ai.Context

	// 1. 通过 SSE 发送压缩中通知
	notifyMsg := fmt.Sprintf("当前上下文超限（%d/%d），正在压缩中...", lastPromptTokens, ctxCfg.Length)
	if err := callback("system", notifyMsg); err != nil {
		logger.Warn("发送压缩通知失败", logger.C(err))
	}

	// 2. 获取压缩技能模板
	skill, err := c.skillManager.Get("context-summarizer")
	if err != nil {
		logger.Warn("获取上下文压缩技能失败，降级为移除最早轮次", logger.C(err))
		return c.fallbackRemoveEarliestRound(mem, callback)
	}

	// 3. 获取压缩模型
	chatModel := c.registry.Get(ctxCfg.Model)
	if chatModel == nil {
		logger.Warn("压缩模型后端不可用，降级为移除最早轮次", logger.S("model", ctxCfg.Model))
		return c.fallbackRemoveEarliestRound(mem, callback)
	}

	// 4. 切分消息：需总结的部分 vs 保留的最近轮次
	recentMsgs := extractRecentRounds(mem.Messages, ctxCfg.RecentRounds)
	oldMsgs := make([]*schema.Message, len(mem.Messages)-len(recentMsgs))
	copy(oldMsgs, mem.Messages[:len(mem.Messages)-len(recentMsgs)])

	// 5. 调用压缩模型生成摘要
	summary, err := c.generateSummary(ctx, chatModel, skill.Content, oldMsgs)
	if err != nil {
		logger.Error("调用压缩模型失败，降级为移除最早轮次", logger.C(err))
		return c.fallbackRemoveEarliestRound(mem, callback)
	}

	// 6. 构建新的消息列表：[系统提示词, 摘要消息, 最近轮次]
	summaryMsg := fmt.Sprintf("【对话历史摘要】\n以下是对早期对话的总结，请结合此摘要和最近对话继续回答用户问题：\n\n%s", summary)
	newMessages := make([]*schema.Message, 0, len(recentMsgs)+2)
	newMessages = append(newMessages, schema.SystemMessage(systemPrompt))
	newMessages = append(newMessages, schema.SystemMessage(summaryMsg))
	newMessages = append(newMessages, recentMsgs...)

	// 7. 原子替换记忆并持久化
	if err := mem.ReplaceMessages(newMessages); err != nil {
		return errorer.NewS(errorer.ErrCompressorReplaceFailed, err)
	}

	logger.Info("上下文压缩完成",
		logger.I("old_msg_count", len(oldMsgs)),
		logger.I("new_msg_count", len(newMessages)),
		logger.I("summary_length", len(summary)),
	)

	return nil
}

// generateSummary 调用压缩模型生成对话摘要（非流式调用）
func (c *Compressor) generateSummary(
	ctx context.Context,
	chatModel model.BaseChatModel,
	skillContent string,
	messages []*schema.Message,
) (string, error) {
	// 将待总结消息序列化为文本
	var convBuilder strings.Builder
	for _, msg := range messages {
		content := msg.Content
		// 截断过长的消息内容，避免摘要请求自身超限
		if len(content) > 8000 {
			content = content[:8000] + "...（内容过长已截断）"
		}
		convBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, content))
	}

	// 构建摘要请求
	summaryMessages := []*schema.Message{
		schema.SystemMessage(skillContent),
		schema.UserMessage(fmt.Sprintf(
			"请总结如下对话历史，保留所有关键信息、决策、待办事项和上下文：\n\n%s",
			convBuilder.String(),
		)),
	}

	resp, err := chatModel.Generate(ctx, summaryMessages)
	if err != nil {
		return "", errorer.NewS(errorer.ErrCompressorCallFailed, err)
	}

	return resp.Content, nil
}

// extractRecentRounds 提取消息列表中最后 n 轮完整对话（以用户消息为轮次边界）
func extractRecentRounds(messages []*schema.Message, n int) []*schema.Message {
	if n <= 0 || len(messages) == 0 {
		return []*schema.Message{}
	}

	// 从后向前遍历，找到第 n 个用户消息的位置
	userCount := 0
	startIdx := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == schema.User {
			userCount++
			if userCount >= n {
				startIdx = i
				break
			}
		}
	}
	return messages[startIdx:]
}

// fallbackRemoveEarliestRound 降级策略：移除最早的1轮对话（第一条用户消息及其助手回复）
func (c *Compressor) fallbackRemoveEarliestRound(
	mem *memory.Memory,
	callback func(event, content string) error,
) error {
	// 找到第一个用户消息的位置
	firstUserIdx := -1
	msgs := mem.Messages
	for i, msg := range msgs {
		if msg.Role == schema.User {
			firstUserIdx = i
			break
		}
	}
	if firstUserIdx == -1 {
		return nil // 没有用户消息，无需移除
	}

	// 找到第二个用户消息的位置（或消息列表末尾）
	nextUserIdx := len(msgs)
	for i := firstUserIdx + 1; i < len(msgs); i++ {
		if msgs[i].Role == schema.User {
			nextUserIdx = i
			break
		}
	}

	// 截断：保留第二个用户消息及其之后的所有消息
	newMessages := msgs[nextUserIdx:]
	if err := mem.ReplaceMessages(newMessages); err != nil {
		return errorer.NewS(errorer.ErrCompressorReplaceFailed, err)
	}

	logger.Info("压缩降级：已移除最早1轮对话",
		logger.I("removed_count", nextUserIdx),
		logger.I("remaining_count", len(newMessages)),
	)

	return nil
}
