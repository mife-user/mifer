package knowledgestore

import (
	"context"
	"fmt"
	"mifer/internal/ai/rag"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// KnowledgeStoreInput 知识库存储输入
type KnowledgeStoreInput struct {
	FilePaths []string `json:"file_paths" jsonschema:"required,description=要存入知识库的文件路径列表（绝对路径或相对路径），文件内容将被分块并向量化存储"`
}

// KnowledgeStoreOutput 知识库存储输出
type KnowledgeStoreOutput struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// New 创建知识库存储工具，通过闭包注入 RAG 服务
func New(ragSvc *rag.Service) (tool.InvokableTool, error) {
	return utils.InferTool("knowledge_store", "将文档文件存入知识库。读取文件内容后自动分块、去重、向量化并存储到向量数据库，供后续检索使用。注意：知识库用于存放文档资料（如技术文档、会议纪要、需求说明等），不要用来存储代码文件——代码文件直接通过 file_reader 读取即可。", func(ctx context.Context, input KnowledgeStoreInput) (KnowledgeStoreOutput, error) {
		if err := ragSvc.Ingest(ctx, input.FilePaths); err != nil {
			return KnowledgeStoreOutput{Error: "存入知识库失败: " + err.Error()}, nil
		}
		return KnowledgeStoreOutput{
			Message: fmt.Sprintf("成功将 %d 个文件存入知识库", len(input.FilePaths)),
		}, nil
	})
}
