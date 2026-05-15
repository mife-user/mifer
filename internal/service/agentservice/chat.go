package agentservice

import (
	"context"
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
		logger.Error("chat failed", logger.C(err))
		return err
	}
	return nil
}
