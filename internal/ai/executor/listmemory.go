package executor

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// ListMemories 返回当前记忆ID及所有可用的记忆ID列表
func (e *Executor) ListMemories(c context.Context) (*domain.MemoryListResp, error) {
	ids, err := e.Humen.Prompt.Memory.ListIDs()
	if err != nil {
		logger.Error(c, "列出记忆失败", logger.C(err))
		return nil, err
	}
	return &domain.MemoryListResp{
		Current: e.Humen.Prompt.Memory.GetCurrentID(),
		IDs:     ids,
	}, nil
}
