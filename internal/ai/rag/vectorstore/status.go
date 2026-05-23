package vectorstore

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	redisv9 "github.com/redis/go-redis/v9"
)

// EnsureIndex 确保 Redis Search 向量索引存在，不存在则创建。
// 索引基于 HASH 类型，PREFIX 匹配 KeyPrefix，包含 content(TEXT) 和 vector_content(VECTOR) 字段。
func EnsureIndex(ctx context.Context, client *redisv9.Client) error {
	ragCfg := conf.GetConfig().Rag
	dim := ragCfg.Dim
	if dim == 0 {
		dim = 768 // nomic-embed-text 默认维度
	}
	indexName := ragCfg.IndexName
	if indexName == "" {
		indexName = "mifer_docs"
	}
	keyPrefix := ragCfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "mifer:docs:"
	}

	// FT.CREATE <index> ON HASH PREFIX 1 <prefix> SCHEMA content TEXT vector_content VECTOR HNSW 6 DIM <dim> DISTANCE_METRIC COSINE

	err := client.Do(ctx, "FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", keyPrefix,
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
		return errorer.NewS(errorer.ErrCreateIndexFailed, err)
	}
	return nil
}
