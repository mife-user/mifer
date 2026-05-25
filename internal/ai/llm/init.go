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
			// default 后端初始化失败为致命错误
			if key == "default" {
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
