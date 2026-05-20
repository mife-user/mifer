package embedder

import (
	"context"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
)

func Init(ctx context.Context, config *conf.Config) (*ark.Embedder, error) {
	emb, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		APIKey:  config.Ai.Backends["embedder"].APIKey,
		BaseURL: config.Ai.Backends["embedder"].BaseURL,
		Model:   config.Ai.Backends["embedder"].Model,
	})
	if err != nil {
		return nil, err
	}
	return emb, nil

}
