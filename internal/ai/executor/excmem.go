package executor

import (
	"context"
	"mifer/internal/domain"
)

func (e *Executor) ExchangeMemory(c context.Context, req *domain.MemoryReq) error {
	e.Humen.Memory.Cfg.Id = req.ID
	return nil
}
