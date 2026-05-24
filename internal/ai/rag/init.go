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
	"mifer/pkg/qdrant"
)

// NewService 创建 RAG 服务，内部初始化 Qdrant 客户端
// 若 Qdrant 不可用则返回 error，调用方应降级为无 RAG 模式
func NewService(ctx context.Context) (*Service, error) {
	// 初始化嵌入模型
	emb, err := embedder.NewEmbedder(ctx)
	if err != nil {
		logger.Error("初始化嵌入模型失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitEmbedderFailed, err)
	}
	// 初始化文件加载器
	fileLoader, err := loader.NewFileLoader(ctx)
	if err != nil {
		logger.Error("初始化文件加载器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitFileLoaderFailed, err)
	}
	// 初始化文本分块器
	chunk, err := chunker.NewChunker(ctx)
	if err != nil {
		logger.Error("初始化文本分块器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitChunkerFailed, err)
	}
	// 初始化 Qdrant 客户端
	qdrantClient, err := qdrant.Init(ctx)
	if err != nil {
		logger.Error("初始化Qdrant客户端失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitQdrantFailed, err)
	}
	// 初始化向量索引器（Qdrant 自动建 Collection）
	idx, err := vectorstore.NewIndexer(ctx, qdrantClient, emb)
	if err != nil {
		logger.Error("初始化向量索引器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitIndexerFailed, err)
	}
	// 初始化向量检索器
	ret, err := vectorstore.NewRetriever(ctx, qdrantClient, emb)
	if err != nil {
		logger.Error("初始化向量检索器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitRetrieverFailed, err)
	}

	ragCfg := conf.GetConfig().Rag
	logger.Info("RAG服务初始化成功",
		logger.S("collection", ragCfg.QdrantCollection),
		logger.S("host", ragCfg.QdrantHost),
		logger.I("port", ragCfg.QdrantPort),
		logger.I("topK", ragCfg.TopK),
	)

	return &Service{
		embedder:  emb,
		loader:    fileLoader,
		chunker:   chunk,
		indexer:   idx,
		retriever: ret,
	}, nil
}
