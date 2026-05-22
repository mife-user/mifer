package embedder

import (
	"context"
	"fmt"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino-ext/components/embedding/ollama"
)

// NewEmbedder 使用 Ollama 创建嵌入器
// 配置来源：config.Ai.Backends["embedder"]，默认 base_url=http://localhost:11434/v1, model=nomic-embed-text
func NewEmbedder(ctx context.Context, config *conf.Config) (*ollama.Embedder, error) {
	backend, ok := config.Ai.Backends["embedder"]
	if !ok {
		return nil, fmt.Errorf("未找到 embedder 后端配置，请在 backends 中配置 embedder")
	}
	if backend.Model == "" {
		return nil, fmt.Errorf("embedder 模型名称为空")
	}

	emb, err := ollama.NewEmbedder(ctx, &ollama.EmbeddingConfig{
		BaseURL: backend.BaseURL,
		Model:   backend.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("创建Ollama嵌入器失败: %w", err)
	}
	return emb, nil
}
