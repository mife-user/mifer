package tui

// memoryListMsg 异步获取记忆列表的结果，由 listMemoriesCmd 发出。
//
// 携带触发上下文（cmd 和 argID），用于判断是展示选择列表还是直接执行命令。
type memoryListMsg struct {
	current string   // 当前记忆ID
	ids     []string // 所有可用记忆ID列表
	err     error    // 网络或解析错误
	cmd     string   // 触发命令："/viewmemory" 或 "/excmem"
	argID   string   // 命令后跟的ID（空表示无ID，需进入选择模式）
}

// memoryViewMsg /viewmemory 加载完成，进入全屏记忆查看模式
type memoryViewMsg struct {
	content string // 格式化的对话记忆文本
	err     error
}

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
