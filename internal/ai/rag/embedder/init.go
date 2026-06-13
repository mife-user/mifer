package embedder

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
)

// NewEmbedder 使用 Ollama 创建嵌入器
// 配置来源：config.Ai.Backends["embedder"]，默认 base_url=http://localhost:11434/v1, model=nomic-embed-text
func NewEmbedder(ctx context.Context) (*ollama.Embedder, error) {
	config := conf.GetConfig()
	backend, ok := config.Ai.Backends["embedder"]
	if !ok {
		return nil, errorer.New(errorer.ErrEmbedderBackendConfig)
	}
	if backend.Model == "" {
		return nil, errorer.New(errorer.ErrEmbedderModelEmpty)
	}

	emb, err := ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		BaseURL: backend.BaseURL,
		Model:   backend.Model,
	})
	if err != nil {
		logger.Error("创建Ollama嵌入器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrCreateEmbedderFailed, err)
	}
	return emb, nil
}
