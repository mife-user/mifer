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

	registry := NewRegistry()

	for key, backendCfg := range backends {
		if backendCfg.Provider == "" {
			logger.Warn(ctx, "后端缺少 provider，跳过", logger.S("backend", key))
			continue
		}
		chatModel, err := registry.initBackend(ctx, key, backendCfg)
		if err != nil {
			if backendCfg.APIKey == "" {
				logger.Warn(ctx, "后端 api_key 未配置，AI 对话功能暂不可用",
					logger.S("backend", key),
					logger.S("provider", backendCfg.Provider))
				continue
			}
			logger.Info(ctx, "初始化后端失败，跳过", logger.S("backend", key), logger.C(err))
			continue
		}
		registry.registerModel(key, chatModel)
	}

	if len(registry.models) > 0 {
		logger.Info(ctx, "模型后端初始化完成", logger.S("backends", fmt.Sprintf("%v", registry.Keys())))
	} else {
		logger.Warn(ctx, "没有成功初始化任何模型后端，AI 对话功能不可用")
	}

	return registry, nil
}
