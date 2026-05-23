package rag

import (
	ollamaembed "github.com/cloudwego/eino-ext/components/embedding/ollama"
	qdrantindexer "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	qdrantretriever "github.com/cloudwego/eino-ext/components/retriever/qdrant"

	"github.com/cloudwego/eino/components/document"
)

// Service RAG 顶层服务，编排嵌入、加载、分块、向量存储、检索全流程
type Service struct {
	embedder  *ollamaembed.Embedder
	loader    document.Loader
	chunker   document.Transformer
	indexer   *qdrantindexer.Indexer
	retriever *qdrantretriever.Retriever
}
