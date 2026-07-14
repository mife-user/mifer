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

// AppendToolExchange 追加一轮工具调用交互到记忆。
// assistantWithToolCall 为包含 ToolCalls 的 Assistant 消息，
// toolResult 为对应的 Tool 结果消息。
// 调用方负责确保 ToolCallID 已正确设置。
func (m *Memory) AppendToolExchange(assistantWithToolCall, toolResult *schema.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, assistantWithToolCall, toolResult)
}
