package rag

import (
	"context"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

// Ingest 文件摄入：加载 → 分块去重 → 向量化存储
func (s *Service) Ingest(ctx context.Context, paths []string) error {
	var allDocs []*schema.Document
	for _, p := range paths {
		docs, err := s.loader.Load(ctx, document.Source{URI: p})
		if err != nil {
			return errorer.NewS(errorer.ErrLoadFileFailed, err)
		}
		allDocs = append(allDocs, docs...)
	}

	chunks, err := s.chunker.Transform(ctx, allDocs)
	if err != nil {
		return errorer.NewS(errorer.ErrChunkProcessFailed, err)
	}

	ids, err := s.indexer.Store(ctx, chunks)
	if err != nil {
		return errorer.NewS(errorer.ErrVectorStoreFailed, err)
	}

	logger.Info("文档摄入完成",
		logger.I("files", len(paths)),
		logger.I("chunks", len(ids)),
	)
	return nil
}
