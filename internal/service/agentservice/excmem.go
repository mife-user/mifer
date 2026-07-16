package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
)

func (s *AgentService) ExchangeMemory(ctx context.Context, req *domain.MemoryReq) error {
	if err := s.Executor.ExchangeMemory(ctx, req); err != nil {
		logger.Error(ctx, "交换记忆失败", logger.C(err))
		return err
	}
	return nil
}
