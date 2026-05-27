package rag

import (
	"context"
	"fmt"
	"mifer/pkg/errorer"
	"sort"
	"strings"

	qdrantretriever "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	qdrant "github.com/qdrant/go-client/qdrant"
)

// Retrieve 检索相关文档
// func (s *Service) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
// 	docs, err := s.retriever.Retrieve(ctx, query)
// 	if err != nil {
// 		return nil, errorer.NewS(errorer.ErrVectorRetrieveFailed, err)
// 	}
// 	return docs, nil
// }

// RetrieveWithContext 检索相关文档并附带上下文分块
// 先语义检索获取匹配分块，再对每个匹配分块查询其前后 contextSize 个相邻分块，合并去重后返回
func (s *Service) RetrieveWithContext(ctx context.Context, query string, contextSize int) ([]*schema.Document, error) {
	matched, err := s.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrVectorRetrieveFailed, err)
	}

	if contextSize <= 0 || len(matched) == 0 {
		return matched, nil
	}

	// 收集已有点的 ID，去重用
	seen := make(map[string]bool, len(matched)*3)
	for _, doc := range matched {
		seen[doc.ID] = true
	}

	result := make([]*schema.Document, len(matched))
	copy(result, matched)

	for _, doc := range matched {
		srcDoc, _ := doc.MetaData["source_document"].(string)
		if srcDoc == "" {
			continue
		}
		chunkIdx, ok := toInt(doc.MetaData["chunk_index"])
		if !ok {
			continue
		}

		gte := float64(chunkIdx - contextSize)
		lte := float64(chunkIdx + contextSize)

		filter := &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatchKeyword("source_document", srcDoc),
				qdrant.NewRange("chunk_index", &qdrant.Range{Gte: &gte, Lte: &lte}),
			},
		}

		neighbors, err := s.retriever.Retrieve(ctx, query,
			qdrantretriever.WithFilter(filter),
			retriever.WithTopK(contextSize*2+1),
		)
		if err != nil {
			continue // 单个文档的邻居查询失败不影响整体结果
		}

		for _, nb := range neighbors {
			if !seen[nb.ID] {
				seen[nb.ID] = true
				result = append(result, nb)
			}
		}
	}

	// 按源文档 + chunk_index 排序
	sort.Slice(result, func(i, j int) bool {
		si, _ := result[i].MetaData["source_document"].(string)
		sj, _ := result[j].MetaData["source_document"].(string)
		if si != sj {
			return si < sj
		}
		ii, _ := toInt(result[i].MetaData["chunk_index"])
		ij, _ := toInt(result[j].MetaData["chunk_index"])
		return ii < ij
	})

	return result, nil
}

// FormatDocs 将检索结果格式化为 Prompt 可用的上下文字符串，按文档分组展示
func (s *Service) FormatDocs(docs []*schema.Document) string {
	if len(docs) == 0 {
		return "暂无相关知识库内容"
	}
	var sb strings.Builder
	var currentDoc string
	docNum := 0
	for _, doc := range docs {
		srcDoc, _ := doc.MetaData["source_document"].(string)
		if srcDoc == "" {
			srcDoc = "未知文档"
		}
		if srcDoc != currentDoc {
			currentDoc = srcDoc
			docNum++
			sb.WriteString(fmt.Sprintf("\n## %s\n", srcDoc))
		}
		chunkIdx, _ := toInt(doc.MetaData["chunk_index"])
		sb.WriteString(fmt.Sprintf("【片段%d】%s\n", chunkIdx, doc.Content))
	}
	_ = docNum
	return sb.String()
}

// toInt 安全地从 interface{} 转换为 int
func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	}
	return 0, false
}
