package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/task"
)

func (s *AgentService) Chat(ctx context.Context, req *domain.TalkReq) (*domain.TalkResp, error) {
	var resp *domain.TalkResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.Chat(ctx, req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
