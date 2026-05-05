package domain

import "context"

type AgentService interface {
	Chat(ctx context.Context, req *TalkReq) (*TalkResp, error)
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
}

type Agent interface {
	Chat(ctx context.Context, req *TalkReq) (*TalkResp, error)
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
}
