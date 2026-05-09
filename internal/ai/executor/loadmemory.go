package executor

import (
	"context"
	"mifer/internal/domain"
)

func (e *Executor) LoadMemory(c context.Context, req *domain.MemoryReq) (*domain.MemoryResp, error) {
	// TODO: 实现从记忆存储中加载指定ID的对话历史
	return &domain.MemoryResp{}, nil
}
