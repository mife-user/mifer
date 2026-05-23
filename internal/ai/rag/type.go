package rag

import (
	ollamaembed "github.com/cloudwego/eino-ext/components/embedding/ollama"

	redisindexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	redisretriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/document"
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
}
