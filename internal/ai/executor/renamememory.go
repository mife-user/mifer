package executor

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// RenameMemory 重命名当前会话记忆。
func (e *Executor) RenameMemory(ctx context.Context, req *domain.RenameMemoryReq) (*domain.RenameMemoryResp, error) {
	oldID := e.Humen.Prompt.Memory.GetCurrentID()
	if err := e.Humen.Prompt.Memory.Rename(req.Name); err != nil {
		logger.Error(ctx, "重命名记忆失败", logger.C(err))
		return nil, err
	}
	return &domain.RenameMemoryResp{OldName: oldID, NewName: req.Name}, nil
}
