package llm

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

type LLM struct {
	Model *openai.ChatModel
}

func Init(c context.Context, config *conf.Config) (*LLM, error) {
	aiCfg := openai.ChatModelConfig{
		Model:   config.Ai.Model,
		BaseURL: config.Ai.BaseURL,
		APIKey:  config.Ai.ApiKey,
	}
	chatModel, err := openai.NewChatModel(c, &aiCfg)
	if err != nil {
		logger.Error("init llm failed", logger.C(err))
		return nil, err
	}
	// 初始化其他LLM服务
	return &LLM{Model: chatModel}, nil
}
