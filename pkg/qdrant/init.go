package qdrant

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	qdrant "github.com/qdrant/go-client/qdrant"
)

// Init 创建 Qdrant gRPC 客户端连接
func Init(ctx context.Context) (*qdrant.Client, error) {
	ragCfg := conf.GetConfig().Rag
	host := ragCfg.QdrantHost
	if host == "" {
		host = "localhost"
	}
	port := ragCfg.QdrantPort
	if port == 0 {
		port = 6334
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   port,
		APIKey:                 ragCfg.QdrantAPIKey,
		SkipCompatibilityCheck: true, // 跳过版本兼容性检查，避免启动时阻塞 60s
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitQdrantFailed, err)
	}

	logger.Info("Qdrant客户端连接成功",
		logger.S("host", host),
		logger.I("port", port),
	)
	return client, nil
}
