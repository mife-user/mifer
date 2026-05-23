package rag

import (
	ollamaembed "github.com/cloudwego/eino-ext/components/embedding/ollama"
	milvus2indexer "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	milvus2retriever "github.com/cloudwego/eino-ext/components/retriever/milvus2"

	"github.com/cloudwego/eino/components/document"
)

// Service RAG 顶层服务，编排嵌入、加载、分块、向量存储、检索全流程
type Service struct {
	embedder  *ollamaembed.Embedder
	loader    document.Loader
	chunker   document.Transformer
	indexer   *milvus2indexer.Indexer
	retriever *milvus2retriever.Retriever
}
