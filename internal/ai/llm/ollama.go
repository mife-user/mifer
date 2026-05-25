package llm

import (
	"context"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/model"
)

// ollamaProvider Ollama 模型提供商
type ollamaProvider struct{}

func (p *ollamaProvider) Name() string { return "ollama" }

func (p *ollamaProvider) InitModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		Model:   cfg.Model,
		BaseURL: baseURL,
	})
}
