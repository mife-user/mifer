package executor

import (
	"context"
	"mifer/internal/domain"
)

func (e *Executor) ExchangeMemory(c context.Context, req *domain.MemoryReq) error {
	return e.Humen.Memory.SwitchSession(req.ID)
}
