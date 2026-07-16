package chunker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// miferNamespace Mifer 项目专用 UUID v5 命名空间，保证同内容在不同项目中产生不同 UUID
var miferNamespace = uuid.MustParse("7b4a5c3d-8e2f-4a1b-9c6d-5f0e8d2a3b1c")

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
		logger.Error(ctx, "创建递归分块器失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrCreateRecursiveChunkerFailed, err)
	}
	return &dedupSplitter{
		splitter: splitter,
	}, nil
}

// Transform 对文档进行分块处理，元数据化 chunk_index 和 source_document
func (d *dedupSplitter) Transform(ctx context.Context, docs []*schema.Document, opts ...document.TransformerOption) ([]*schema.Document, error) {
	chunks, err := d.splitter.Transform(ctx, docs, opts...)
	if err != nil {
		logger.Error(ctx, "文档切分失败", logger.C(err))
		return nil, err
	}

	// 按源文档分组，计算每个文档的分块索引
	docGroups := make(map[string][]int) // sourceDoc -> chunk positions
	for i, ch := range chunks {
		docGroups[ch.ID] = append(docGroups[ch.ID], i)
	}

	seen := make(map[string]bool, len(chunks))
	result := make([]*schema.Document, 0, len(chunks))

	// 为每个源文档独立编号，避免重复分块
	for sourceDoc, positions := range docGroups {
		docIdx := 0
		for _, pos := range positions {
			ch := chunks[pos]
			hash := sha256.Sum256([]byte(ch.Content))
			hashStr := fmt.Sprintf("%x", hash)[:16]
			if seen[hashStr] {
				continue
			}
			seen[hashStr] = true

			ch.ID = uuid.NewSHA1(miferNamespace, []byte(sourceDoc+"\x00"+ch.Content)).String()
			if ch.MetaData == nil {
				ch.MetaData = make(map[string]any)
			}
			// 元数据化 chunk_index 和 source_document
			ch.MetaData["source_document"] = sourceDoc
			ch.MetaData["chunk_index"] = docIdx
			ch.MetaData["chunk_hash"] = hashStr
			ch.MetaData["total_chunks"] = len(positions)
			docIdx++
			result = append(result, ch)
		}
	}
	return result, nil
}
