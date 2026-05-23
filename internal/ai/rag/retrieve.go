package rag

import (
	"context"
	"fmt"
	"mifer/pkg/errorer"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// Retrieve 检索相关文档
func (s *Service) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
	docs, err := s.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrVectorRetrieveFailed, err)
	}
	return docs, nil
}

// FormatDocs 将检索结果格式化为 Prompt 可用的上下文字符串
func (s *Service) FormatDocs(docs []*schema.Document) string {
	if len(docs) == 0 {
		return "暂无相关知识库内容"
	}
	var sb strings.Builder
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("【文档%d】%s\n", i+1, doc.Content))
	}
	return sb.String()
}
