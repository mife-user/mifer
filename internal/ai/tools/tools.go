package tools

import (
	"mifer/internal/ai/rag"
	"mifer/internal/ai/tools/commandexecutor"
	"mifer/internal/ai/tools/filecreator"
	"mifer/internal/ai/tools/filereader"
	"mifer/internal/ai/tools/fileviewer"
	"mifer/internal/ai/tools/filewriter"
	"mifer/internal/ai/tools/imagegenerator"
	"mifer/internal/ai/tools/knowledgesearch"
	"mifer/internal/ai/tools/knowledgestore"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// FileTools 返回文件操作相关工具（读取、写入、创建、查看、图片生成）
func FileTools(mmModel model.BaseChatModel) []tool.BaseTool {
	var tools []tool.BaseTool

	fr, err := filereader.New()
	if err != nil {
		logger.Error("创建 file_reader 工具失败", logger.C(err))
	} else {
		tools = append(tools, fr)
	}

	fw, err := filewriter.New()
	if err != nil {
		logger.Error("创建 file_writer 工具失败", logger.C(err))
	} else {
		tools = append(tools, fw)
	}

	fc, err := filecreator.New()
	if err != nil {
		logger.Error("创建 file_creator 工具失败", logger.C(err))
	} else {
		tools = append(tools, fc)
	}

	fv, err := fileviewer.New(mmModel)
	if err != nil {
		logger.Error("创建 file_viewer 工具失败", logger.C(err))
	} else {
		tools = append(tools, fv)
	}

	ig, err := imagegenerator.New(mmModel)
	if err != nil {
		logger.Error("创建 image_generator 工具失败", logger.C(err))
	} else {
		tools = append(tools, ig)
	}

	return tools
}

// CommandTools 返回命令执行相关工具（需传入 config 以注入安全策略）
func CommandTools() []tool.BaseTool {
	var tools []tool.BaseTool

	ce, err := commandexecutor.New()
	if err != nil {
		logger.Error("创建 command_executor 工具失败", logger.C(err))
	} else {
		tools = append(tools, ce)
	}

	return tools
}

// AuditTools 返回安全审计相关工具（文件读取、文件查看）
func AuditTools(mmModel model.BaseChatModel) []tool.BaseTool {
	var tools []tool.BaseTool

	fr, err := filereader.New()
	if err != nil {
		logger.Error("创建 file_reader 工具失败", logger.C(err))
	} else {
		tools = append(tools, fr)
	}

	fv, err := fileviewer.New(mmModel)
	if err != nil {
		logger.Error("创建 file_viewer 工具失败", logger.C(err))
	} else {
		tools = append(tools, fv)
	}

	return tools
}

// KnowledgeTools 返回知识库相关工具（检索 + 存储），ragSvc 为 nil 时返回空切片
func KnowledgeTools(ragSvc rag.RAGService) []tool.BaseTool {
	var tools []tool.BaseTool
	if ragSvc == nil {
		return tools
	}

	ks, err := knowledgesearch.New(ragSvc)
	if err != nil {
		logger.Error("创建 knowledge_search 工具失败", logger.C(err))
	} else {
		tools = append(tools, ks)
	}

	kst, err := knowledgestore.New(ragSvc)
	if err != nil {
		logger.Error("创建 knowledge_store 工具失败", logger.C(err))
	} else {
		tools = append(tools, kst)
	}

	return tools
}

