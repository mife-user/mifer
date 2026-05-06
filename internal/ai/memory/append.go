package memory

import (
	"github.com/cloudwego/eino/schema"
)

// Append 添加一条用户消息到记忆中
func (m *Memory) Append(content string) []*schema.Message {
	m.Messages = append(m.Messages, schema.UserMessage(content))
	return m.Messages
}
