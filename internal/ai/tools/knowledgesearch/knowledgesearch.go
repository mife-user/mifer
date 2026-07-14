package knowledgesearch

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// KnowledgeSearchInput 知识库检索输入
type KnowledgeSearchInput struct {
	Query       string `json:"query" jsonschema:"required,description=搜索查询文本，用于在知识库中检索相关文档"`
	ContextSize int    `json:"context_size" jsonschema:"description=上下文窗口大小，检索匹配分块时同时返回其前后各N个相邻分块，默认0表示不扩展"`
}

// KnowledgeSearchOutput 知识库检索输出
type KnowledgeSearchOutput struct {
	Results string `json:"results"`
	Count   int    `json:"count"`
	Error   string `json:"error,omitempty"`
}

// retriever 知识检索接口，仅包含本工具所需的方法。
// rag.RAGService 接口实现了此接口。
type retriever interface {
	RetrieveWithContext(ctx context.Context, query string, contextSize int) ([]*schema.Document, error)
	FormatDocs(docs []*schema.Document) string
}

// New 创建知识库检索工具，通过闭包注入实现了 retriever 接口的服务。
func New(ragSvc retriever) (tool.InvokableTool, error) {
	return utils.InferTool("knowledge_search", "检索知识库中的相关文档内容。当你不确定某个知识点或需要查找已有文档中的信息时，使用此工具搜索知识库。注意：知识库存放的是文档资料，不是代码——如需查看代码文件请使用 file_reader。", func(ctx context.Context, input KnowledgeSearchInput) (KnowledgeSearchOutput, error) {
		docs, err := ragSvc.RetrieveWithContext(ctx, input.Query, input.ContextSize)
		if err != nil {
			return KnowledgeSearchOutput{Error: "检索知识库失败: " + err.Error()}, nil
		}
		return KnowledgeSearchOutput{
			Results: ragSvc.FormatDocs(docs),
			Count:   len(docs),
		}, nil
	})
}
