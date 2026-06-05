package agentservice

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// Compact 手动触发上下文压缩
func (s *AgentService) Compact(ctx context.Context) (*domain.CompactResp, error) {
	var resp *domain.CompactResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.Compact(ctx)
		return err
	})
	if err != nil {
		logger.Error("手动压缩上下文失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
