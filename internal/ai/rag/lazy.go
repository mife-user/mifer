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

	"github.com/cloudwego/eino/schema"
)

// NewLazyService 创建懒加载 RAG 服务，仅初始化无网络组件（embedder/loader/chunker），
// Qdrant 连接推迟到首次 Retrieve 或 Ingest 调用时
func NewLazyService(ctx context.Context) *LazyService {
	emb, err := embedder.NewEmbedder(ctx)
	if err != nil {
		logger.Error("初始化嵌入模型失败，RAG服务不可用", logger.C(err))
		return &LazyService{initErr: err}
	}

	fileLoader, err := loader.NewFileLoader(ctx)
	if err != nil {
		logger.Error("初始化文件加载器失败，RAG服务不可用", logger.C(err))
		return &LazyService{initErr: err}
	}

	chunk, err := chunker.NewChunker(ctx)
	if err != nil {
		logger.Error("初始化文本分块器失败，RAG服务不可用", logger.C(err))
		return &LazyService{initErr: err}
	}

	return &LazyService{
		embedder: emb,
		loader:   fileLoader,
		chunker:  chunk,
	}
}

// ensureReady 确保 RAG 底层服务已初始化，首次调用时连接 Qdrant 并创建 indexer/retriever
// 使用 Mutex 而非 sync.Once，支持失败后重试
func (s *LazyService) ensureReady(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 已成功初始化
	if s.svc != nil {
		return nil
	}

	// 无网络组件初始化失败，不可恢复
	if s.initErr != nil && s.embedder == nil {
		return s.initErr
	}

	// 尝试连接 Qdrant 并构建完整 Service
	ragCfg := conf.GetConfig().Rag
	qdrantCfg := qdrant.QdrantCfg{
		Host:   ragCfg.QdrantHost,
		Port:   ragCfg.QdrantPort,
		APIKey: ragCfg.QdrantAPIKey,
	}
	qdrantClient, err := qdrantCfg.Init(ctx)
	if err != nil {
		s.initErr = errorer.New(errorer.ErrRAGNotReady)
		logger.Warn("Qdrant客户端连接失败，知识库暂时不可用", logger.C(err))
		return errorer.New(errorer.ErrRAGNotReady)
	}

	idx, err := vectorstore.NewIndexer(ctx, qdrantClient, s.embedder)
	if err != nil {
		s.initErr = errorer.New(errorer.ErrRAGNotReady)
		logger.Warn("向量索引器初始化失败，知识库暂时不可用", logger.C(err))
		return errorer.New(errorer.ErrRAGNotReady)
	}

	ret, err := vectorstore.NewRetriever(ctx, qdrantClient, s.embedder)
	if err != nil {
		s.initErr = errorer.New(errorer.ErrRAGNotReady)
		logger.Warn("向量检索器初始化失败，知识库暂时不可用", logger.C(err))
		return errorer.New(errorer.ErrRAGNotReady)
	}

	s.svc = &Service{
		embedder:     s.embedder,
		loader:       s.loader,
		chunker:      s.chunker,
		indexer:      idx,
		retriever:    ret,
		qdrantClient: qdrantClient,
	}
	s.initErr = nil
	logger.Info("RAG知识库服务已就绪")
	return nil
}

// Retrieve 检索相关文档（首次调用时触发懒初始化）
// func (s *LazyService) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
// 	if err := s.ensureReady(ctx); err != nil {
// 		return nil, err
// 	}
// 	return s.svc.Retrieve(ctx, query)
// }

// RetrieveWithContext 检索相关文档并附带上下文分块（首次调用时触发懒初始化）
func (s *LazyService) RetrieveWithContext(ctx context.Context, query string, contextSize int) ([]*schema.Document, error) {
	if err := s.ensureReady(ctx); err != nil {
		return nil, err
	}
	return s.svc.RetrieveWithContext(ctx, query, contextSize)
}

// FormatDocs 将检索结果格式化为上下文字符串，委托给底层 Service
func (s *LazyService) FormatDocs(docs []*schema.Document) string {
	if s.svc != nil {
		return s.svc.FormatDocs(docs)
	}
	// 懒加载未完成时仍可格式化空结果
	return "知识库服务尚未就绪，暂无检索结果"
}

// Ingest 文件摄入（首次调用时触发懒初始化）
func (s *LazyService) Ingest(ctx context.Context, paths []string) error {
	if err := s.ensureReady(ctx); err != nil {
		return err
	}
	return s.svc.Ingest(ctx, paths)
}
