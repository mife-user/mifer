package memory

import (
	"github.com/cloudwego/eino/schema"
)

// AppendUser 添加一条用户消息到记忆中
func (m *Memory) AppendUser(content string) []*schema.Message {
	m.Messages = append(m.Messages, schema.UserMessage(content))
	return m.Messages
}

// AppendAssistant 添加一条助手消息到记忆中
func (m *Memory) AppendAssistant(content string) []*schema.Message {
	m.Messages = append(m.Messages, schema.AssistantMessage(content, nil))
	return m.Messages
}
