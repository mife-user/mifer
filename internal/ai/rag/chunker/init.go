package chunker

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/schema"
)

// ChunkerConfig 分块配置
type ChunkerConfig struct {
	ChunkSize    int
	ChunkOverlap int
}

// NewChunker 创建文本分块器：递归分块 + SHA256 去重
func NewChunker(ctx context.Context, cfg *ChunkerConfig) (document.Transformer, error) {
	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   cfg.ChunkSize,
		OverlapSize: cfg.ChunkOverlap,
	})
	if err != nil {
		return nil, fmt.Errorf("创建递归分块器失败: %w", err)
	}
	return &dedupSplitter{
		splitter: splitter,
	}, nil
}

// dedupSplitter 在递归分块后对内容做 SHA256 去重，并为每条分块生成唯一 ID
type dedupSplitter struct {
	splitter document.Transformer
}

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

func (d *dedupSplitter) GetType() string {
	return "DedupRecursiveSplitter"
}
