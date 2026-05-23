package vectorstore

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	redisindexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	redisretriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/embedding"
	redisv9 "github.com/redis/go-redis/v9"
)

// NewIndexer 基于 Eino 官方 Redis Indexer 创建向量索引器
func NewIndexer(ctx context.Context, client *redisv9.Client, emb embedding.Embedder) (*redisindexer.Indexer, error) {
	keyPrefix := conf.GetConfig().Rag.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "mifer:docs:"
	}
	idx, err := redisindexer.NewIndexer(ctx, &redisindexer.IndexerConfig{
		Client:    client,
		KeyPrefix: keyPrefix,
		Embedding: emb,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateIndexFailed, err)
	}
	return idx, nil
}

// NewRetriever 基于 Eino 官方 Redis Retriever 创建向量检索器
func NewRetriever(ctx context.Context, client *redisv9.Client, emb embedding.Embedder) (*redisretriever.Retriever, error) {
	ragCfg := conf.GetConfig().Rag
	indexName := ragCfg.IndexName
	if indexName == "" {
		indexName = "mifer_docs"
	}
	topK := ragCfg.TopK
	if topK == 0 {
		topK = 5
	}
	r, err := redisretriever.NewRetriever(ctx, &redisretriever.RetrieverConfig{
		Client:    client,
		Index:     indexName,
		TopK:      topK,
		Embedding: emb,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateRetrieverFailed, err)
	}
	return r, nil
}
