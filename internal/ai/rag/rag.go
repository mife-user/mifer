package rag

import (
	"context"
	"fmt"
	"mifer/internal/ai/rag/chunker"
	"mifer/internal/ai/rag/embedder"
	"mifer/internal/ai/rag/loader"
	ollamaembed "github.com/cloudwego/eino-ext/components/embedding/ollama"
	"mifer/internal/ai/rag/vectorstore"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"mifer/pkg/res"
	"strings"

	"github.com/cloudwego/eino/components/document"
	redisindexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	redisretriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/schema"
	redisv9 "github.com/redis/go-redis/v9"
)

// Service RAG 顶层服务，编排嵌入、加载、分块、向量存储、检索全流程
type Service struct {
	embedder  *ollamaembed.Embedder
	loader    document.Loader
	chunker   document.Transformer
	indexer   *redisindexer.Indexer
	retriever *redisretriever.Retriever
	client    *redisv9.Client
	config    *RAGConfig
}

// NewService 创建 RAG 服务，内部初始化 Redis 客户端
// 若 Redis 不可用则返回 error，调用方应降级为无 RAG 模式
func NewService(ctx context.Context, config *conf.Config) (*Service, error) {
	ragCfg := DefaultRAGConfig()

	emb, err := embedder.NewEmbedder(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("初始化嵌入器失败: %w", err)
	}

	fileLoader, err := loader.NewFileLoader(ctx)
	if err != nil {
		return nil, fmt.Errorf("初始化文件加载器失败: %w", err)
	}

	chunk, err := chunker.NewChunker(ctx, &chunker.ChunkerConfig{
		ChunkSize:    ragCfg.ChunkSize,
		ChunkOverlap: ragCfg.ChunkOverlap,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化分块器失败: %w", err)
	}

	redisClient, err := res.Init(config)
	if err != nil {
		return nil, fmt.Errorf("初始化Redis失败: %w", err)
	}

	storeCfg := &vectorstore.StoreConfig{
		KeyPrefix: ragCfg.KeyPrefix,
		IndexName: ragCfg.IndexName,
		TopK:      ragCfg.TopK,
		Dim:       ragCfg.Dim,
	}

	if err := vectorstore.EnsureIndex(ctx, redisClient, storeCfg); err != nil {
		return nil, fmt.Errorf("初始化向量索引失败: %w", err)
	}

	idx, err := vectorstore.NewIndexer(ctx, redisClient, emb, storeCfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Indexer失败: %w", err)
	}

	ret, err := vectorstore.NewRetriever(ctx, redisClient, emb, storeCfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Retriever失败: %w", err)
	}

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
		config:    ragCfg,
	}, nil
}

// Ingest 文件摄入：加载 → 分块去重 → 向量化存储
func (s *Service) Ingest(ctx context.Context, paths []string) error {
	var allDocs []*schema.Document
	for _, p := range paths {
		docs, err := s.loader.Load(ctx, document.Source{URI: p})
		if err != nil {
			return fmt.Errorf("加载文件失败 [%s]: %w", p, err)
		}
		allDocs = append(allDocs, docs...)
	}

	chunks, err := s.chunker.Transform(ctx, allDocs)
	if err != nil {
		return fmt.Errorf("分块处理失败: %w", err)
	}

	ids, err := s.indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("向量存储失败: %w", err)
	}

	logger.Info("文档摄入完成",
		logger.I("files", len(paths)),
		logger.I("chunks", len(ids)),
	)
	return nil
}

// Retrieve 检索相关文档
func (s *Service) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
	docs, err := s.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}
	return docs, nil
}

// FormatDocs 将检索结果格式化为 Prompt 可用的上下文字符串
func FormatDocs(docs []*schema.Document) string {
	if len(docs) == 0 {
		return "暂无相关知识库内容"
	}
	var sb strings.Builder
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("【文档%d】%s\n", i+1, doc.Content))
	}
	return sb.String()
}

// Close 关闭 Redis 连接
func (s *Service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
