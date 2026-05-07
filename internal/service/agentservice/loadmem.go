package agentservice

import (
	"context"
	"mifer/internal/domain"
)

func (s *AgentService) LoadMemory(ctx context.Context, req *domain.MemoryReq) (*domain.MemoryResp, error) {
	return &domain.MemoryResp{}, nil
}
