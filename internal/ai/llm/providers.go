package llm

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/model"
)

// initBackend 根据 key 和配置初始化后端模型
func (r *Registry) initBackend(ctx context.Context, key string, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	provider, ok := r.providers[cfg.Provider]
	if !ok {
		return nil, errorer.NewF(errorer.ErrUnsupportedProvider, cfg.Provider, key)
	}
	logger.Info(ctx, "初始化模型后端", logger.S("backend", key), logger.S("provider", cfg.Provider), logger.S("model", cfg.Model))
	return provider.InitModel(ctx, cfg)
}
