package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

func (s *AgentService) LoadMemory(ctx context.Context, req *domain.MemoryReq) (*domain.MemoryResp, error) {
	var resp *domain.MemoryResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.LoadMemory(ctx, req)
		return err
	})
	if err != nil {
		logger.Error(ctx, "加载记忆失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
