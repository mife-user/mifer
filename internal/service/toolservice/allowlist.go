package toolservice

import (
	"context"
	"slices"

	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// AddAllowList 添加命令到白名单。
// 先检查命令是否已存在，避免重复添加；然后写入持久化文件。
func (s *ToolService) AddAllowList(ctx context.Context, req *domain.ToolAddAllowListReq) (*domain.ToolAddAllowListResp, error) {
	var resp *domain.ToolAddAllowListResp
	err := task.Do(ctx, func() error {
		// 检查是否已在白名单中
		existing, err := conf.LoadAllowList()
		if err != nil {
			logger.Warn(ctx, "加载命令白名单失败", logger.C(err))
		}
		if err == nil && slices.Contains(existing, req.Command) {
			resp = &domain.ToolAddAllowListResp{
				Command: req.Command,
				Added:   false,
			}
			return nil
		}

		// 添加命令到白名单文件
		if err := conf.AddToAllowList(s.workdir, req.Command); err != nil {
			logger.Error(ctx, "添加命令到白名单失败",
				logger.S("command", req.Command),
				logger.C(err))
			return err
		}

		logger.Info(ctx, "命令已添加到白名单", logger.S("command", req.Command))
		resp = &domain.ToolAddAllowListResp{
			Command: req.Command,
			Added:   true,
		}
		return nil
	})
	if err != nil {
		logger.Error(ctx, "添加白名单失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
