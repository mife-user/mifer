package executor

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// ClearMemory 生成新记忆会话ID并切换过去
func (e *Executor) ClearMemory(ctx context.Context) (*domain.ClearMemoryResp, error) {
	newID, err := e.Humen.Prompt.Memory.GenerateID()
	if err != nil {
		logger.Error(ctx, "生成记忆ID失败", logger.C(err))
		return nil, err
	}
	if err := e.Humen.Prompt.Memory.SwitchSession(newID); err != nil {
		logger.Error(ctx, "切换记忆会话失败", logger.C(err))
		return nil, err
	}
	return &domain.ClearMemoryResp{NewID: newID}, nil
}
