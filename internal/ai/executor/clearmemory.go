package executor

import (
	"context"
	"mifer/internal/domain"
)

// ClearMemory 生成新记忆会话ID并切换过去
func (e *Executor) ClearMemory(ctx context.Context) (*domain.ClearMemoryResp, error) {
	newID, err := e.Humen.Prompt.Memory.GenerateID()
	if err != nil {
		return nil, err
	}
	if err := e.Humen.Prompt.Memory.SwitchSession(newID); err != nil {
		return nil, err
	}
	return &domain.ClearMemoryResp{NewID: newID}, nil
}
