package agentservice

import (
	"context"
	"errors"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

func (s *AgentService) Chat(ctx context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	var err error
	err = task.Do(ctx, func() error {
		err = s.Executor.Chat(ctx, req, callback)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Warn(ctx, "chat canceled（客户端断开）", logger.C(err))
			return nil
		}
		logger.Error(ctx, "chat failed", logger.C(err))
		return err
	}
	return nil
}
