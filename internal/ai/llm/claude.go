package llm

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/components/model"
)

// claudeProvider Claude 模型提供商
type claudeProvider struct{}

func (p *claudeProvider) Name() string { return "claude" }

func (p *claudeProvider) InitModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errorer.New(errorer.ErrApiKey)
	}
	return claude.NewChatModel(ctx, &claude.Config{
		Model:     cfg.Model,
		APIKey:    cfg.APIKey,
		MaxTokens: 4096,
	})
}
