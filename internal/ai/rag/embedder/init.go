package embedder

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
)

// NewEmbedder 使用 Ollama 创建嵌入器
// 自动从 ai.backends 中找到第一个 type=embedding 的后端配置
func NewEmbedder(ctx context.Context) (*ollama.Embedder, error) {
	config := conf.GetConfig()

	// 查找第一个 type=embedding 的后端
	var backend conf.BackendConfig
	found := false
	for _, cfg := range config.Ai.Backends {
		if cfg.Type == "embedding" {
			backend = cfg
			found = true
			break
		}
	}
	if !found {
		return nil, errorer.New(errorer.ErrNoEmbeddingBackend)
	}
	if backend.Model == "" {
		return nil, errorer.New(errorer.ErrEmbedderModelEmpty)
	}

	emb, err := ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		BaseURL: backend.BaseURL,
		Model:   backend.Model,
	})
	if err != nil {
		logger.Error(ctx, "创建Ollama嵌入器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrCreateEmbedderFailed, err)
	}
	return emb, nil
}
