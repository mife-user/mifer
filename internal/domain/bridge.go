package domain

import "context"

type AgentService interface {
	Chat(ctx context.Context, req *TalkReq, callback func(event, content string) error) error
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
	ExchangeMemory(ctx context.Context, req *MemoryReq) error
	ListMemories(ctx context.Context) (*MemoryListResp, error)
	ClearMemory(ctx context.Context) (*ClearMemoryResp, error)
	ListRebackEntries(ctx context.Context) (*RebackListResp, error)
	Reback(ctx context.Context, req *RebackReq) (*RebackResp, error)
	GetPrompt(ctx context.Context) (*PromptResp, error)
	SetPrompt(ctx context.Context, req *PromptReq) (*PromptResp, error)
	ResetPrompt(ctx context.Context) (*PromptResp, error)
	ListPlans(ctx context.Context) (*PlanListResp, error)
	LoadPlan(ctx context.Context, name string) (*PlanLoadResp, error)
	MCPStatus(ctx context.Context) (*MCPStatusResp, error)
	ListSkills(ctx context.Context) (*SkillListResp, error)
	Compact(ctx context.Context) (*CompactResp, error)
	ListAgents(ctx context.Context) (*AgentListResp, error)
}

type Agent interface {
	Chat(ctx context.Context, req *TalkReq, callback func(event, content string) error) error
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
	ExchangeMemory(ctx context.Context, req *MemoryReq) error
	ListMemories(ctx context.Context) (*MemoryListResp, error)
	ClearMemory(ctx context.Context) (*ClearMemoryResp, error)
	ListRebackEntries(ctx context.Context) (*RebackListResp, error)
	Reback(ctx context.Context, req *RebackReq) (*RebackResp, error)
	GetPrompt(ctx context.Context) (*PromptResp, error)
	SetPrompt(ctx context.Context, req *PromptReq) (*PromptResp, error)
	ResetPrompt(ctx context.Context) (*PromptResp, error)
	ListPlans(ctx context.Context) (*PlanListResp, error)
	LoadPlan(ctx context.Context, name string) (*PlanLoadResp, error)
	MCPStatus(ctx context.Context) (*MCPStatusResp, error)
	ListSkills(ctx context.Context) (*SkillListResp, error)
	Compact(ctx context.Context) (*CompactResp, error)
	ListAgents(ctx context.Context) (*AgentListResp, error)
}

// ToolService 工具操作服务接口，按 Handler → Service → Executor 三层架构定义。
// 实现方放在 internal/service/toolservice/ 包中。
type ToolService interface {
	Confirm(ctx context.Context, req *ToolConfirmReq) (*ToolConfirmResp, error)
	AddAllowList(ctx context.Context, req *ToolAddAllowListReq) (*ToolAddAllowListResp, error)
}
