package llm

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

// geminiProvider Gemini 模型提供商
type geminiProvider struct{}

func (p *geminiProvider) Name() string { return "gemini" }

func (p *geminiProvider) InitModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errorer.New(errorer.ErrApiKey)
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateGeminiClientFailed, err)
	}
	return gemini.NewChatModel(ctx, &gemini.Config{
		Client: client,
		Model:  cfg.Model,
	})
}
