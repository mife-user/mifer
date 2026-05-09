package domain

import "context"

type AgentService interface {
	Chat(ctx context.Context, req *TalkReq, callback func(content string) error) error
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
}

type Agent interface {
	Chat(ctx context.Context, req *TalkReq, callback func(content string) error) error
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
}
