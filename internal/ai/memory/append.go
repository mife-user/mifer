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

// AppendMessages 批量追加消息到记忆中，用于持久化工具调用消息等中间消息。
// 调用方负责保证消息顺序正确（assistant+ToolCalls 后跟 tool 结果）。
// 不触发持久化，需调用方随后调用 Save()。
func (m *Memory) AppendMessages(msgs []*schema.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msgs...)
}
