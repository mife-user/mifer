package memory

import "github.com/cloudwego/eino/schema"

// CountUserMessages 统计记忆中当前已保存的用户消息数量，即当前对话轮次。
// 用于在每轮对话结束时获取当前轮次号。
func (m *Memory) CountUserMessages() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, msg := range m.Messages {
		if msg.Role == schema.User {
			count++
		}
	}
	return count
}
