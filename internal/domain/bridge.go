package domain

import "context"

type AgentService interface {
	Chat(ctx context.Context, req *TalkReq, callback func(event, content string) error) error
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
	ExchangeMemory(ctx context.Context, req *MemoryReq) error
	ListMemories(ctx context.Context) (*MemoryListResp, error)
}

type Agent interface {
	Chat(ctx context.Context, req *TalkReq, callback func(event, content string) error) error
	LoadMemory(ctx context.Context, req *MemoryReq) (*MemoryResp, error)
	ExchangeMemory(ctx context.Context, req *MemoryReq) error
	ListMemories(ctx context.Context) (*MemoryListResp, error)
}
