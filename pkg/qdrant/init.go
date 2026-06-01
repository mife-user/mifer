package qdrant

import (
	"context"

	qdrant "github.com/qdrant/go-client/qdrant"
)

type QdrantCfg struct {
	Host   string
	Port   int
	APIKey string
}

// Init 创建 Qdrant gRPC 客户端连接
func (c *QdrantCfg) Init(ctx context.Context) (*qdrant.Client, error) {
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	port := c.Port
	if port == 0 {
		port = 6334
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   port,
		APIKey:                 c.APIKey,
		SkipCompatibilityCheck: true, // 跳过版本兼容性检查，避免启动时阻塞 60s
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
