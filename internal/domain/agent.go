package domain

type TalkReq struct {
	Content string
}

type MemoryReq struct {
	ID string
}

type MemoryResp struct {
	Memory string
}

type MemoryListResp struct {
	Current string   // 当前记忆ID
	IDs     []string // 所有可用记忆ID列表
}

type ClearMemoryResp struct {
	NewID string // 新生成的记忆会话ID
}

type PromptReq struct {
	Prompt string // 系统提示词文本
}

type PromptResp struct {
	Prompt string // 当前生效的系统提示词
}
