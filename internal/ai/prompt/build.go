package prompt

import "github.com/cloudwego/eino/schema"

// Build 组装完整 prompt：prepend 系统提示 + Memory 中的对话历史
// 返回给 Runner.Run() 使用的消息列表
func (p *Prompty) Build() []*schema.Message {
	if p.SystemPrompt == "" {
		return p.Memory.Messages
	}
	msgs := make([]*schema.Message, 0, len(p.Memory.Messages)+1)
	msgs = append(msgs, schema.SystemMessage(p.SystemPrompt))
	msgs = append(msgs, p.Memory.Messages...)
	return msgs
}
