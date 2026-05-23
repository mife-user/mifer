package vectorstore

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	qdrantindexer "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	qdrantretriever "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/embedding"
	qdrant "github.com/qdrant/go-client/qdrant"
)

// NewIndexer 基于 Eino Qdrant Indexer 创建向量索引器
func NewIndexer(ctx context.Context, client *qdrant.Client, emb embedding.Embedder) (*qdrantindexer.Indexer, error) {
	ragCfg := conf.GetConfig().Rag
	collection := ragCfg.QdrantCollection
	if collection == "" {
		collection = "mifer_docs"
	}
	dim := ragCfg.Dim
	if dim == 0 {
		dim = 768
	}
	idx, err := qdrantindexer.NewIndexer(ctx, &qdrantindexer.Config{
		Client:     client,
		Collection: collection,
		VectorDim:  dim,
		Distance:   qdrant.Distance_Cosine,
		Embedding:  emb,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateIndexFailed, err)
	}
	return idx, nil
}

// NewRetriever 基于 Eino Qdrant Retriever 创建向量检索器
func NewRetriever(ctx context.Context, client *qdrant.Client, emb embedding.Embedder) (*qdrantretriever.Retriever, error) {
	ragCfg := conf.GetConfig().Rag
	collection := ragCfg.QdrantCollection
	if collection == "" {
		collection = "mifer_docs"
	}
	topK := ragCfg.TopK
	if topK == 0 {
		topK = 5
	}
	r, err := qdrantretriever.NewRetriever(ctx, &qdrantretriever.Config{
		Client:     client,
		Collection: collection,
		TopK:       topK,
		Embedding:  emb,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateRetrieverFailed, err)
	}
	return r, nil
}
