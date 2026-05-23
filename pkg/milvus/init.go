package milvus

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Init 创建 Milvus gRPC 客户端连接
func Init(ctx context.Context) (*milvusclient.Client, error) {
	ragCfg := conf.GetConfig().Rag
	addr := ragCfg.MilvusAddress
	if addr == "" {
		addr = "localhost:19530"
	}

	client, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address:  addr,
		Username: ragCfg.MilvusUsername,
		Password: ragCfg.MilvusPassword,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitMilvusFailed, err)
	}

	logger.Info("Milvus客户端连接成功", logger.S("address", addr))
	return client, nil
}
