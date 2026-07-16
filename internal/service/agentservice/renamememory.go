package agentservice

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// RenameMemory 重命名当前会话记忆。
func (s *AgentService) RenameMemory(ctx context.Context, req *domain.RenameMemoryReq) (*domain.RenameMemoryResp, error) {
	var resp *domain.RenameMemoryResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.RenameMemory(ctx, req)
		return err
	})
	if err != nil {
		logger.Error(ctx, "重命名记忆失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
