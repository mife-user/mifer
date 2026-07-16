package compressor

import (
	"context"
	"fmt"
	"strings"

	"mifer/internal/ai/memory"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

const (
	maxToolResultLen = 50000 // Layer2 中 ToolResult 截断阈值（字符数）
)

// Compress 执行三层上下文压缩：
//   Layer 1（最近 recentRounds 轮）— 完整保留
//   Layer 2（中间 slimRounds 轮）— 保留 ToolCall + 截断/offload ToolResult
//   Layer 3（更早轮次）— 调用压缩模型生成摘要
//
// 降级策略：技能缺失、模型不可用或调用失败时，移除最早的轮次。
func (c *Compressor) Compress(
	ctx context.Context,
	mem *memory.Memory,
	systemPrompt string,
	lastPromptTokens int,
	callback func(event, content string) error,
) error {
	ctxCfg := conf.GetConfig().Ai.Context

	notifyMsg := fmt.Sprintf("当前上下文超限（%d/%d），正在压缩中...", lastPromptTokens, ctxCfg.Length)
	if err := callback("system", notifyMsg); err != nil {
		logger.Warn(ctx, "发送压缩通知失败", logger.C(err))
	}

	allMsgs := mem.Messages()

	// 切分轮次：以 User 消息为边界
	rounds := splitRounds(allMsgs)
	if len(rounds) <= c.recentRounds {
		// 消息总轮数不足，跳过压缩
		logger.Debug(ctx, "上下文轮次不足，跳过压缩")
		return nil
	}

	// 计算各层边界
	layer1Start := len(rounds) - c.recentRounds
	if layer1Start < 0 {
		layer1Start = 0
	}
	layer2Start := layer1Start - c.slimRounds
	if layer2Start < 0 {
		layer2Start = 0
	}

	layer3Rounds := rounds[:layer2Start]  // 需压缩为摘要
	layer2Rounds := rounds[layer2Start:layer1Start] // 需精简处理
	layer1Rounds := rounds[layer1Start:]  // 完整保留

	logger.Debug(ctx, "三层压缩边界",
		logger.I("total_rounds", len(rounds)),
		logger.I("layer3_summarize", len(layer3Rounds)),
		logger.I("layer2_slim", len(layer2Rounds)),
		logger.I("layer1_keep", len(layer1Rounds)),
	)

	var newMessages []*schema.Message

	// 处理 Layer 3: 生成摘要
	if len(layer3Rounds) > 0 {
		layer3Msgs := flattenRounds(layer3Rounds)
		summary, err := c.summarizeLayer3(ctx, systemPrompt, layer3Msgs)
		if err != nil {
			logger.Error(ctx, "Layer3 摘要生成失败，降级为移除最早轮次", logger.C(err))
			return c.fallbackRemoveEarliestRounds(mem, rounds, c.recentRounds)
		}
		newMessages = append(newMessages, summary...)
	}

	// 处理 Layer 2: 精简（保留 ToolCall + 截断/offload ToolResult）
	for _, round := range layer2Rounds {
		slimmed := c.slimRound(round)
		newMessages = append(newMessages, slimmed...)
	}

	// 处理 Layer 1: 完整保留
	for _, round := range layer1Rounds {
		newMessages = append(newMessages, round...)
	}

	// 原子替换记忆并持久化
	if err := mem.ReplaceMessages(newMessages); err != nil {
		return errorer.NewS(errorer.ErrCompressorReplaceFailed, err)
	}

	logger.Info(ctx, "上下文压缩完成",
		logger.I("layer3_count", len(layer3Rounds)),
		logger.I("layer2_count", len(layer2Rounds)),
		logger.I("layer1_count", len(layer1Rounds)),
		logger.I("total_messages", len(newMessages)),
	)

	return nil
}

// ──────────────────────────── 轮次切分 ────────────────────────────

// splitRounds 以 User 消息为边界切分消息列表为若干轮次。
// 每个轮次从一条 User 消息开始，包含后续所有 Assistant + Tool 消息，
// 直到下一条 User 消息之前。
func splitRounds(msgs []*schema.Message) [][]*schema.Message {
	if len(msgs) == 0 {
		return nil
	}

	var rounds [][]*schema.Message
	var current []*schema.Message

	for _, msg := range msgs {
		if msg.Role == schema.User && len(current) > 0 {
			rounds = append(rounds, current)
			current = nil
		}
		current = append(current, msg)
	}

	// 前置的非 User 消息（如 system prompt + 历史摘要）作为首个轮次
	if len(current) > 0 {
		rounds = append(rounds, current)
	}

	return rounds
}

// flattenRounds 将多个轮次展平为单一消息切片。
func flattenRounds(rounds [][]*schema.Message) []*schema.Message {
	var result []*schema.Message
	for _, r := range rounds {
		result = append(result, r...)
	}
	return result
}

// ──────────────────────────── Layer 3: 摘要 ────────────────────────────

// summarizeLayer3 调用压缩模型将早期对话压缩为一条系统消息摘要。
func (c *Compressor) summarizeLayer3(
	ctx context.Context,
	systemPrompt string,
	messages []*schema.Message,
) ([]*schema.Message, error) {
	skill, err := c.skillManager.Get("context-summarizer")
	if err != nil {
		return nil, fmt.Errorf("获取摘要技能失败: %w", err)
	}

	ctxCfg := conf.GetConfig().Ai.Context
	chatModel := c.registry.Get(ctxCfg.Backend)
	if chatModel == nil {
		return nil, fmt.Errorf("压缩模型 [%s] 不可用", ctxCfg.Backend)
	}

	// 构建摘要请求
	var convBuilder strings.Builder
	for _, msg := range messages {
		content := msg.Content
		const maxBytes = 8000
		if len(content) > maxBytes {
			cut := maxBytes
			for cut > 0 && cut > maxBytes-4 {
				if content[cut]&0xC0 != 0x80 {
					break
				}
				cut--
			}
			content = content[:cut] + "...（内容过长已截断）"
		}
		// 标注 tool 调用信息
		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				convBuilder.WriteString(fmt.Sprintf("[%s]: 调用工具 %s(%s)\n", msg.Role, tc.Function.Name, tc.Function.Arguments))
			}
		} else if msg.Role == schema.Tool {
			convBuilder.WriteString(fmt.Sprintf("[%s/%s]: %s\n", msg.Role, msg.ToolName, content))
		} else {
			convBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, content))
		}
	}

	summaryMessages := []*schema.Message{
		schema.SystemMessage(skill.Content),
		schema.UserMessage(fmt.Sprintf(
			"请总结如下对话历史，保留所有关键信息、决策、待办事项和工具调用记录：\n\n%s",
			convBuilder.String(),
		)),
	}

	resp, err := chatModel.Generate(ctx, summaryMessages)
	if err != nil {
		return nil, fmt.Errorf("调用压缩模型失败: %w", err)
	}

	summaryContent := fmt.Sprintf("【对话历史摘要】\n以下是对早期对话的总结，请结合此摘要和最近对话继续回答用户问题。\n\n%s", resp.Content)

	result := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.SystemMessage(summaryContent),
	}

	return result, nil
}

// ──────────────────────────── Layer 2: 精简 ────────────────────────────

// slimRound 精简单个轮次：保留 ToolCall 原文 + 截断/offload 大 ToolResult。
func (c *Compressor) slimRound(round []*schema.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(round))
	for _, msg := range round {
		if msg.Role == schema.Tool && len(msg.Content) > maxToolResultLen {
			result = append(result, c.offloadToolResult(msg))
		} else {
			result = append(result, msg)
		}
	}
	return result
}

// offloadToolResult 将超长工具结果 offload 到文件，返回带占位符的 Tool 消息。
func (c *Compressor) offloadToolResult(msg *schema.Message) *schema.Message {
	// 深拷贝消息，避免修改原始切片中的引用
	copied := *msg
	copied.Extra = nil

	originalContent := msg.Content

	// 生成 offload 文件键
	key := fmt.Sprintf("%s/%s.txt", msg.ToolName, msg.ToolCallID)
	if msg.ToolCallID == "" {
		key = fmt.Sprintf("%s/%s_%d.txt", msg.ToolName, msg.ToolName, len(originalContent))
	}

	filePath, err := c.offloader.Save(context.Background(), key, []byte(originalContent))
	if err != nil {
		logger.Warn(context.Background(), "Offload 工具结果失败，保留截断原文", logger.S("tool", msg.ToolName), logger.C(err))
		// 降级：截断保留前 maxToolResultLen 字符
		if len(originalContent) > maxToolResultLen {
			copied.Content = originalContent[:maxToolResultLen] + fmt.Sprintf("\n\n...（已截断，原长度 %d 字符）", len(originalContent))
		}
		return &copied
	}

	// 替换为占位符
	copied.Content = fmt.Sprintf(
		"【工具结果已存档】工具 [%s] 返回了约 %d 字符的结果，完整内容已保存至 %s。"+
			"如需查看完整结果，请使用 file_reader 工具读取该文件。\n\n"+
			"结果摘要（前 1000 字符）：\n%s",
		msg.ToolName, len(originalContent), filePath,
		truncateContent(originalContent, 1000),
	)

	logger.Debug(context.Background(), "已 offload 超长工具结果",
		logger.S("tool", msg.ToolName),
		logger.S("callID", msg.ToolCallID),
		logger.S("offloadPath", filePath),
		logger.I("originalLen", len(originalContent)),
	)

	return &copied
}

// ──────────────────────────── 降级策略 ────────────────────────────

// fallbackRemoveEarliestRounds 降级：保留最后 keepRounds 轮，丢弃更早轮次。
func (c *Compressor) fallbackRemoveEarliestRounds(
	mem *memory.Memory,
	rounds [][]*schema.Message,
	keepRounds int,
) error {
	if len(rounds) <= keepRounds {
		return nil
	}

	kept := rounds[len(rounds)-keepRounds:]
	newMessages := flattenRounds(kept)
	if err := mem.ReplaceMessages(newMessages); err != nil {
		return errorer.NewS(errorer.ErrCompressorReplaceFailed, err)
	}

	logger.Info(context.Background(), "压缩降级：已移除最早轮次",
		logger.I("removed_rounds", len(rounds)-keepRounds),
		logger.I("remaining_rounds", keepRounds),
	)

	return nil
}

// ──────────────────────────── 工具函数 ────────────────────────────

// truncateContent 按 UTF-8 字符边界截断内容，返回前 maxLen 个字节。
func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := maxLen
	for cut > 0 && cut > maxLen-4 {
		if s[cut]&0xC0 != 0x80 {
			break
		}
		cut--
	}
	return s[:cut] + "..."
}
