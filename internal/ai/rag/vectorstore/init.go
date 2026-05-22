package vectorstore

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	redisindexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	redisretriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	redisv9 "github.com/redis/go-redis/v9"
)

// StoreConfig 向量存储配置
type StoreConfig struct {
	KeyPrefix string
	IndexName string
	TopK      int
	Dim       int // 向量维度，默认 768（nomic-embed-text）
}

// NewIndexer 基于 Eino 官方 Redis Indexer 创建向量索引器
func NewIndexer(ctx context.Context, client *redisv9.Client, emb embedding.Embedder, cfg *StoreConfig) (*redisindexer.Indexer, error) {
	idx, err := redisindexer.NewIndexer(ctx, &redisindexer.IndexerConfig{
		Client:    client,
		KeyPrefix: cfg.KeyPrefix,
		Embedding: emb,
	})
	if err != nil {
		return nil, fmt.Errorf("创建Redis Indexer失败: %w", err)
	}
	return idx, nil
}

// NewRetriever 基于 Eino 官方 Redis Retriever 创建向量检索器
func NewRetriever(ctx context.Context, client *redisv9.Client, emb embedding.Embedder, cfg *StoreConfig) (*redisretriever.Retriever, error) {
	r, err := redisretriever.NewRetriever(ctx, &redisretriever.RetrieverConfig{
		Client:    client,
		Index:     cfg.IndexName,
		TopK:      cfg.TopK,
		Embedding: emb,
	})
	if err != nil {
		return nil, fmt.Errorf("创建Redis Retriever失败: %w", err)
	}
	return r, nil
}

// EnsureIndex 确保 Redis Search 向量索引存在，不存在则创建。
// 索引基于 HASH 类型，PREFIX 匹配 KeyPrefix，包含 content(TEXT) 和 vector_content(VECTOR) 字段。
func EnsureIndex(ctx context.Context, client *redisv9.Client, cfg *StoreConfig) error {
	dim := cfg.Dim
	if dim == 0 {
		dim = 768 // nomic-embed-text 默认维度
	}

	// FT.CREATE <index> ON HASH PREFIX 1 <prefix> SCHEMA content TEXT vector_content VECTOR HNSW 6 DIM <dim> DISTANCE_METRIC COSINE
	createCmd := fmt.Sprintf(
		"FT.CREATE %s ON HASH PREFIX 1 %s SCHEMA content TEXT vector_content VECTOR HNSW 6 DIM %d DISTANCE_METRIC COSINE",
		cfg.IndexName, cfg.KeyPrefix, dim,
	)

	err := client.Do(ctx, "FT.CREATE", cfg.IndexName,
		"ON", "HASH",
		"PREFIX", "1", cfg.KeyPrefix,
		"SCHEMA",
		"content", "TEXT",
		"vector_content", "VECTOR", "HNSW", "6",
		"DIM", dim,
		"DISTANCE_METRIC", "COSINE",
	).Err()
	if err != nil {
		// 索引已存在不算错误
		if err.Error() == "Index already exists" {
			return nil
		}
		return fmt.Errorf("创建Redis向量索引失败 [%s]: %w", createCmd, err)
	}
	return nil
}
