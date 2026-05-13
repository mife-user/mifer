package agentservice

import (
	"context"
	"mifer/internal/domain"
)

func (s *AgentService) ExchangeMemory(ctx context.Context, req *domain.MemoryReq) error {
	if err := s.Executor.ExchangeMemory(ctx, req); err != nil {
		return err
	}
	return nil
}
