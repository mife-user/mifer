package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// ClearMemory 生成新记忆会话ID并切换
func (s *AgentService) ClearMemory(ctx context.Context) (*domain.ClearMemoryResp, error) {
	var resp *domain.ClearMemoryResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ClearMemory(ctx)
		return err
	})
	if err != nil {
		logger.Error(ctx, "清空记忆失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
