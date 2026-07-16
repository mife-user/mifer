package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// ListPlans 列出所有可用的计划文件
func (s *AgentService) ListPlans(ctx context.Context) (*domain.PlanListResp, error) {
	var resp *domain.PlanListResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ListPlans(ctx)
		return err
	})
	if err != nil {
		logger.Error(ctx, "列出计划文件失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}

// LoadPlan 加载指定计划文件内容
func (s *AgentService) LoadPlan(ctx context.Context, name string) (*domain.PlanLoadResp, error) {
	var resp *domain.PlanLoadResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.LoadPlan(ctx, name)
		return err
	})
	if err != nil {
		logger.Error(ctx, "加载计划文件失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
