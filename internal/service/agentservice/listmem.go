package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// ListMemories 列出所有可用记忆ID及当前记忆ID
func (s *AgentService) ListMemories(ctx context.Context) (*domain.MemoryListResp, error) {
	var resp *domain.MemoryListResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ListMemories(ctx)
		return err
	})
	if err != nil {
		logger.Error("列出记忆列表失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
