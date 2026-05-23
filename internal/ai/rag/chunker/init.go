package chunker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

// NewChunker 创建文本分块器：递归分块 + SHA256 去重
func NewChunker(ctx context.Context) (document.Transformer, error) {
	ragCfg := conf.GetConfig().Rag
	chunkSize := ragCfg.ChunkSize
	if chunkSize == 0 {
		chunkSize = 500
	}
	chunkOverlap := ragCfg.ChunkOverlap
	if chunkOverlap == 0 {
		chunkOverlap = 50
	}
	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   chunkSize,
		OverlapSize: chunkOverlap,
	})
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateRecursiveChunkerFailed, err)
	}
	return &dedupSplitter{
		splitter: splitter,
	}, nil
}

// Transform 对文档进行分块处理
func (d *dedupSplitter) Transform(ctx context.Context, docs []*schema.Document, opts ...document.TransformerOption) ([]*schema.Document, error) {
	chunks, err := d.splitter.Transform(ctx, docs, opts...)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(chunks))
	result := make([]*schema.Document, 0, len(chunks))

	for i, ch := range chunks {
		hash := sha256.Sum256([]byte(ch.Content))
		hashStr := fmt.Sprintf("%x", hash)[:16]
		if seen[hashStr] {
			continue
		}
		seen[hashStr] = true

		ch.ID = fmt.Sprintf("%s_chunk_%d_%s", ch.ID, i, hashStr)
		if ch.MetaData == nil {
			ch.MetaData = make(map[string]any)
		}
		ch.MetaData["chunk_hash"] = hashStr
		result = append(result, ch)
	}
	return result, nil
}
