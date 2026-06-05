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

// PlanListResp 计划文件列表响应
type PlanListResp struct {
	Files []string // .mifer/plans/ 目录下的文件名列表
}

// MCPServerStatus MCP Server 状态
type MCPServerStatus struct {
	Name      string // Server 名称
	Status    string // connected / disabled / error / disconnected
	ToolCount int    // 已加载的工具数量
	Error     string // 错误信息
}

// SkillInfo 技能简要信息
type SkillInfo struct {
	Name        string // 技能名
	Description string // 技能描述
	Context     string // 执行模式: inline / fork
	Agent       string // fork 模式的目标 Agent 名
}

// SkillListResp 技能列表响应
type SkillListResp struct {
	Skills []SkillInfo
}

// MCPStatusResp MCP 状态响应
type MCPStatusResp struct {
	Servers []MCPServerStatus
}

// PlanLoadResp 计划文件内容响应
type PlanLoadResp struct {
	Name    string // 文件名
	Content string // 文件内容
}

// CompactResp 上下文压缩结果
type CompactResp struct {
	Message string // 结果描述消息
}
