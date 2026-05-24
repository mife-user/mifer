package rag

import (
	"context"
	"sync"

	ollamaembed "github.com/cloudwego/eino-ext/components/embedding/ollama"
	qdrantindexer "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	qdrantretriever "github.com/cloudwego/eino-ext/components/retriever/qdrant"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

// RAGService 定义 RAG 服务接口，Service 和 LazyService 均实现此接口
type RAGService interface {
	Retrieve(ctx context.Context, query string) ([]*schema.Document, error)
	FormatDocs(docs []*schema.Document) string
	Ingest(ctx context.Context, paths []string) error
}

// Service RAG 顶层服务，编排嵌入、加载、分块、向量存储、检索全流程
type Service struct {
	embedder  *ollamaembed.Embedder
	loader    document.Loader
	chunker   document.Transformer
	indexer   *qdrantindexer.Indexer
	retriever *qdrantretriever.Retriever
}

// LazyService RAG 懒加载服务，构造时仅初始化无网络组件，首次调用时延迟连接 Qdrant
type LazyService struct {
	svc     *Service
	mu      sync.Mutex
	initErr error

	embedder *ollamaembed.Embedder
	loader   document.Loader
	chunker  document.Transformer
}
