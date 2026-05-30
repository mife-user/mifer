package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// ListSkills 返回所有已加载的技能列表
func (s *AgentService) ListSkills(ctx context.Context) (*domain.SkillListResp, error) {
	var resp *domain.SkillListResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ListSkills(ctx)
		return err
	})
	if err != nil {
		logger.Error("获取技能列表失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
