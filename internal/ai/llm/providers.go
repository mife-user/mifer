package llm

import (
	"context"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

func initOpenAIModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errorer.New(errorer.ErrApiKey)
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
	})
}

func initClaudeModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errorer.New(errorer.ErrApiKey)
	}
	return claude.NewChatModel(ctx, &claude.Config{
		Model:     cfg.Model,
		APIKey:    cfg.APIKey,
		MaxTokens: 4096,
	})
}

func initGeminiModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	if cfg.APIKey == "" {
		return nil, errorer.New(errorer.ErrApiKey)
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini 客户端失败: %w", err)
	}
	return gemini.NewChatModel(ctx, &gemini.Config{
		Client: client,
		Model:  cfg.Model,
	})
}

func initOllamaModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		Model:   cfg.Model,
		BaseURL: baseURL,
	})
}

// providerInitMap 映射 provider 名称到初始化函数
var providerInitMap = map[string]func(context.Context, conf.BackendConfig) (model.BaseChatModel, error){
	"openai": initOpenAIModel,
	"claude": initClaudeModel,
	"gemini": initGeminiModel,
	"ollama": initOllamaModel,
}

func initBackend(ctx context.Context, key string, cfg conf.BackendConfig) (model.BaseChatModel, error) {
	initFn, ok := providerInitMap[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("不支持的模型提供商: %s（后端: %s），支持: openai, claude, gemini, ollama", cfg.Provider, key)
	}
	logger.Info("初始化模型后端", logger.S("backend", key), logger.S("provider", cfg.Provider), logger.S("model", cfg.Model))
	return initFn(ctx, cfg)
}
