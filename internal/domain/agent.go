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

// Reback
type RebackReq struct {
	Index int // 回退到的用户消息轮次索引（1-based）
}

type RebackEntry struct {
	Index   int    // 用户消息轮次索引
	Summary string // 用户消息摘要
}

type RebackListResp struct {
	Entries []RebackEntry // 可回退的对话轮次列表
}

type RebackResp struct {
	Summary string // 被删除的用户消息摘要
	Content string // 被删除的用户消息完整内容（用于预填输入框）
	Message string // 回退结果描述
}

type PromptResp struct {
	Prompt string // 当前生效的系统提示词
}
