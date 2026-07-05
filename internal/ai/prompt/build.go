package prompt

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Build 组装完整 prompt：模板变量替换 → 返回消息列表
// query 为用户本轮输入，作为模板中 {query} 变量的值
func (p *Prompty) Build(ctx context.Context, query string) ([]*schema.Message, error) {
	// 对历史消息消毒：移除孤立 tool 消息，防止 LLM API 400 错误
	history := sanitizeMessages(p.Memory.Messages())

	// 若未配置模板，回退到手工拼接（保持向后兼容）
	if p.Template == nil {
		return p.buildLegacyWithHistory(query, history), nil
	}

	return p.Template.Format(ctx, map[string]any{
		"system_prompt": p.SystemPrompt,
		"history":       history,
		"query":         query,
	})
}

// buildLegacy 手工拼接消息，用于模板不可用时的回退路径
func (p *Prompty) buildLegacy(query string) []*schema.Message {
	return p.buildLegacyWithHistory(query, sanitizeMessages(p.Memory.Messages()))
}

// buildLegacyWithHistory 使用已消毒的历史消息手工拼接 prompt
func (p *Prompty) buildLegacyWithHistory(query string, history []*schema.Message) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+2)
	if p.SystemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(p.SystemPrompt))
	}
	msgs = append(msgs, history...)
	msgs = append(msgs, schema.UserMessage(query))
	return msgs
}
