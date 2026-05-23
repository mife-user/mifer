package rag

import (
	"context"
	"mifer/internal/ai/rag/chunker"
	"mifer/internal/ai/rag/embedder"
	"mifer/internal/ai/rag/loader"
	"mifer/internal/ai/rag/vectorstore"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"mifer/pkg/res"
)

// NewService 创建 RAG 服务，内部初始化 Redis 客户端
// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
func NewService(ctx context.Context) (*Service, error) {
	// 初始化嵌入模型
	// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
	emb, err := embedder.NewEmbedder(ctx)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitEmbedderFailed, err)
	}
	// 初始化文件加载器
	fileLoader, err := loader.NewFileLoader(ctx)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitFileLoaderFailed, err)
	}
	// 初始化文本分块器
	chunk, err := chunker.NewChunker(ctx)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitChunkerFailed, err)
	}
	// 初始化 Redis 客户端
	// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
	redisClient, err := res.Init()
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitRedisFailed, err)
	}
	// 确保 Redis Search 向量索引存在，不存在则创建
	// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
	if err := vectorstore.EnsureIndex(ctx, redisClient); err != nil {
		return nil, errorer.NewS(errorer.ErrEnsureIndexFailed, err)
	}
	// 初始化向量索引器
	// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
	idx, err := vectorstore.NewIndexer(ctx, redisClient, emb)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitIndexerFailed, err)
	}
	// 初始化向量检索器
	// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
	ret, err := vectorstore.NewRetriever(ctx, redisClient, emb)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrInitRetrieverFailed, err)
	}
	// 初始化 RAG 服务
	// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
	ragCfg := conf.GetConfig().Rag
	logger.Info("RAG服务初始化成功",
		logger.S("index", ragCfg.IndexName),
		logger.I("topK", ragCfg.TopK),
	)

	return &Service{
		embedder:  emb,
		loader:    fileLoader,
		chunker:   chunk,
		indexer:   idx,
		retriever: ret,
		client:    redisClient,
	}, nil
}
