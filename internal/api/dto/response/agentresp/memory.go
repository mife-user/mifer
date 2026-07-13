package agentres

// MemoryRes 加载记忆响应
type MemoryRes struct {
	Memory string `json:"memory"`
}

// MemoryListRes 记忆列表响应
type MemoryListRes struct {
	Current string   `json:"current"`
	IDs     []string `json:"ids"`
}

// ExchangeMemoryRes 记忆交换响应
type ExchangeMemoryRes struct {
	Message string `json:"message"`
}

type ClearMemoryRes struct {
	NewID string `json:"new_id"`
}

// RenameMemoryRes 重命名记忆响应
type RenameMemoryRes struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
	Message string `json:"message"`
}

// CompactRes 上下文压缩结果响应
type CompactRes struct {
	Message string `json:"message"`
}
