package llm

import (
	"context"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
)

// InitRegistry 根据配置初始化所有模型后端
func InitRegistry(ctx context.Context) (*Registry, error) {
	backends := conf.GetConfig().Ai.Backends
	if len(backends) == 0 {
		return nil, errorer.New(errorer.ErrNoBackendConfig)
	}

	// default 后端必须存在
	if _, ok := backends["default"]; !ok {
		return nil, errorer.New(errorer.ErrDefaultBackendConfig)
	}

	registry := NewRegistry()

	for key, backendCfg := range backends {
		if backendCfg.Provider == "" {
			logger.Warn("后端缺少 provider，跳过", logger.S("backend", key))
			continue
		}
		chatModel, err := registry.initBackend(ctx, key, backendCfg)
		if err != nil {
			// default 后端因 api_key 缺失导致初始化失败时，不阻止启动
			// 程序将在运行时通过 /api/admin/status 提示用户配置
			if key == "default" {
				if backendCfg.APIKey == "" {
					logger.Warn("默认后端api_key未配置，AI对话功能暂不可用，请在/config中设置api_key",
						logger.S("provider", backendCfg.Provider))
					continue
				}
				return nil, err
			}
			logger.Info("初始化后端失败，跳过", logger.S("backend", key), logger.C(err))
			continue
		}
		registry.models[key] = chatModel
	}

	logger.Info("模型后端初始化完成", logger.S("backends", fmt.Sprintf("%v", registry.Keys())))

	return registry, nil
}
