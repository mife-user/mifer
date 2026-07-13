package executor

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/conf"
)

// BackendStatus 返回当前各后端的加载状态，供 TUI 启动时检查模型就绪状态
func (e *Executor) BackendStatus(ctx context.Context) (*domain.BackendStatusResp, error) {
	cfg := conf.GetConfig()
	resp := &domain.BackendStatusResp{
		Ready:    e.Humen.Registry.IsReady(),
		Backends: make([]domain.BackendStatusEntry, 0),
		Warnings: make([]string, 0),
	}

	loadedKeys := e.Humen.Registry.Keys()
	loadedSet := make(map[string]bool, len(loadedKeys))
	for _, k := range loadedKeys {
		loadedSet[k] = true
	}

	for key, backendCfg := range cfg.Ai.Backends {
		// embedding 类型的后端不属于聊天模型，跳过状态展示
		if backendCfg.Type == "embedding" {
			continue
		}
		entry := domain.BackendStatusEntry{
			Name:  key,
			Model: backendCfg.Model,
		}
		if loadedSet[key] {
			entry.Status = "ok"
		} else {
			if backendCfg.APIKey == "" {
				entry.Status = "not_configured"
				entry.Error = "api_key 未配置，请在 /config 中设置"
			} else {
				entry.Status = "failed"
				entry.Error = "后端初始化失败，请检查 provider、base_url、api_key 配置"
			}
		}
		resp.Backends = append(resp.Backends, entry)
	}

	// 收集警告信息 — 检查是否有任何 chat 后端缺 api_key
	hasMissingKey := false
	for _, backendCfg := range cfg.Ai.Backends {
		if backendCfg.Type != "embedding" && backendCfg.APIKey == "" {
			hasMissingKey = true
			break
		}
	}
	if hasMissingKey {
		resp.Warnings = append(resp.Warnings, "当前部分后端 api_key 未配置，请输入 /config 配置")
	}

	return resp, nil
}
