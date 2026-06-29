package prompt

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Build 组装完整 prompt：模板变量替换 → 返回消息列表
// query 为用户本轮输入，作为模板中 {query} 变量的值
func (p *Prompty) Build(ctx context.Context, query string) ([]*schema.Message, error) {
	// 若未配置模板，回退到手工拼接（保持向后兼容）
	if p.Template == nil {
		return p.buildLegacy(query), nil
	}

	return p.Template.Format(ctx, map[string]any{
		"system_prompt": p.SystemPrompt,
		"history":       p.Memory.Messages(),
		"query":         query,
	})
}

// buildLegacy 手工拼接消息，用于模板不可用时的回退路径
func (p *Prompty) buildLegacy(query string) []*schema.Message {
	memMsgs := p.Memory.Messages()
	msgs := make([]*schema.Message, 0, len(memMsgs)+2)
	if p.SystemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(p.SystemPrompt))
	}
	msgs = append(msgs, memMsgs...)
	msgs = append(msgs, schema.UserMessage(query))
	return msgs
}
