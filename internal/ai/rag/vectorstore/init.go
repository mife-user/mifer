package vectorstore

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	milvus2indexer "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	milvus2retriever "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// NewIndexer 基于 Eino Milvus2 Indexer 创建向量索引器
func NewIndexer(ctx context.Context, client *milvusclient.Client, emb embedding.Embedder) (*milvus2indexer.Indexer, error) {
	ragCfg := conf.GetConfig().Rag
	collection := ragCfg.MilvusCollection
	if collection == "" {
		collection = "mifer_docs"
	}
	dim := ragCfg.Dim
	if dim == 0 {
		dim = 768
	}
	idx, err := milvus2indexer.NewIndexer(ctx, &milvus2indexer.IndexerConfig{
		Client:     client,
		Collection: collection,
		Vector: &milvus2indexer.VectorConfig{
			Dimension:  int64(dim),
			MetricType: milvus2indexer.COSINE,
		},
		Embedding: emb,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateIndexFailed, err)
	}
	return idx, nil
}

// NewRetriever 基于 Eino Milvus2 Retriever 创建向量检索器
func NewRetriever(ctx context.Context, client *milvusclient.Client, emb embedding.Embedder) (*milvus2retriever.Retriever, error) {
	ragCfg := conf.GetConfig().Rag
	collection := ragCfg.MilvusCollection
	if collection == "" {
		collection = "mifer_docs"
	}
	topK := ragCfg.TopK
	if topK == 0 {
		topK = 5
	}
	r, err := milvus2retriever.NewRetriever(ctx, &milvus2retriever.RetrieverConfig{
		Client:     client,
		Collection: collection,
		TopK:       topK,
		SearchMode: search_mode.NewApproximate(milvus2retriever.COSINE),
		Embedding:  emb,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateRetrieverFailed, err)
	}
	return r, nil
}
