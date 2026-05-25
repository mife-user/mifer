package llm

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// openAIProvider OpenAI 模型提供商
type openAIProvider struct{}

func (p *openAIProvider) Name() string { return "openai" }

func (p *openAIProvider) InitModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errorer.New(errorer.ErrApiKey)
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
	})
}
