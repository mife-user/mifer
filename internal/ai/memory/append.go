package memory

import (
	"github.com/cloudwego/eino/schema"
)

// AppendUser 添加一条用户消息到记忆中
func (m *Memory) AppendUser(content string) []*schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, schema.UserMessage(content))
	return m.messages
}

// AppendAssistant 添加一条助手消息到记忆中
func (m *Memory) AppendAssistant(content string) []*schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, schema.AssistantMessage(content, nil))
	return m.messages
}
