package toolservice

import (
	"context"

	"mifer/internal/ai/confirm"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// Confirm 处理工具确认请求。
// 将 action 字符串映射为 ConfirmResult，查找待确认项并写入结果。
func (s *ToolService) Confirm(ctx context.Context, req *domain.ToolConfirmReq) (*domain.ToolConfirmResp, error) {
	var resp *domain.ToolConfirmResp
	err := task.Do(ctx, func() error {
		// 映射 action 字符串到 ConfirmResult
		var result confirm.ConfirmResult
		switch req.Action {
		case "confirm", "allow":
			result = confirm.ConfirmResult{Approved: true, Action: req.Action}
		case "deny":
			result = confirm.ConfirmResult{Approved: false, Action: "deny"}
		default:
			logger.Warn("无效的确认动作", logger.S("action", req.Action))
			resp = &domain.ToolConfirmResp{ID: req.ID, Resolved: false, Action: req.Action}
			return nil
		}

		entry, ok := s.store.Get(req.ID)
		if !ok {
			logger.Warn("确认条目未找到或已过期", logger.S("id", req.ID))
			resp = &domain.ToolConfirmResp{ID: req.ID, Resolved: false, Action: req.Action}
			return nil
		}

		s.store.Resolve(req.ID, result)

		logger.Info("工具确认已处理",
			logger.S("id", req.ID),
			logger.S("action", req.Action),
			logger.S("tool", entry.ToolName))

		resp = &domain.ToolConfirmResp{
			ID:       req.ID,
			Resolved: true,
			Action:   req.Action,
		}
		return nil
	})
	if err != nil {
		logger.Error("处理工具确认失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
