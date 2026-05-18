package tui

// ============================================================================
// memoryItem — list.Item 实现，用于记忆选择列表
// ============================================================================

// memoryItem 实现 bubbles/list.Item 接口，表示一条可选的记忆会话。
type memoryItem struct {
	id      string // 记忆ID
	current bool   // 是否为当前会话
}

func (i memoryItem) Title() string { return i.id }
func (i memoryItem) Description() string {
	if i.current {
		return "(当前)"
	}
	return ""
}
func (i memoryItem) FilterValue() string { return i.id }
