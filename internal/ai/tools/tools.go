package tools

import (
	"mifer/internal/ai/rag"
	"mifer/internal/ai/tools/fileviewer"
	"mifer/internal/ai/tools/imagegenerator"
	"mifer/internal/ai/tools/knowledgesearch"
	"mifer/internal/ai/tools/knowledgestore"
	qqtools "mifer/internal/ai/tools/qq"
	"mifer/internal/ai/tools/webfetch"
	"mifer/internal/ai/tools/websearch"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"mifer/qq"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// FileTools 返回文件操作相关工具（读取、写入、创建、查看、图片生成）
func Image(mmModel model.BaseChatModel) []tool.BaseTool {
	var tools []tool.BaseTool
	ig, err := imagegenerator.New(mmModel)
	if err != nil {
		logger.Error("创建 image_generator 工具失败", logger.C(err))
	} else {
		tools = append(tools, ig)
	}
	ig, err = fileviewer.New(mmModel)
	if err != nil {
		logger.Error("创建 file_viewer 工具失败", logger.C(err))
	} else {
		tools = append(tools, ig)
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

// WebTools 返回网页相关工具（搜索 + 抓取）
func WebTools() []tool.BaseTool {
	var tools []tool.BaseTool

	ws, err := websearch.New()
	if err != nil {
		logger.Error("创建 web_search 工具失败", logger.C(err))
	} else {
		tools = append(tools, ws)
	}

	wf, err := webfetch.New()
	if err != nil {
		logger.Error("创建 web_fetch 工具失败", logger.C(err))
	} else {
		tools = append(tools, wf)
	}

	return tools
}

// QQTools 返回 QQ 消息相关工具（发送消息等）。
// getSender 延迟获取 qq.Sender 实现，避免工具构造时 Sender 尚未初始化。
func QQTools(getSender func() qq.Sender) []tool.BaseTool {
	var tools []tool.BaseTool

	qs, err := qqtools.NewSendMessage(getSender)
	if err != nil {
		logger.Error("创建 qq_send_message 工具失败", logger.C(err))
	} else {
		tools = append(tools, qs)
	}

	return tools
}

func NewWithName(name []string, mmModel model.BaseChatModel, ragSvc rag.RAGService) ([]tool.BaseTool, error) {
	var tools []tool.BaseTool
	for _, n := range name {
		switch n {
		case "image_generator":
			ig, err := imagegenerator.New(mmModel)
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, ig)
		case "knowledge_search":
			ks, err := knowledgesearch.New(ragSvc)
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, ks)
		case "knowledge_store":
			kst, err := knowledgestore.New(ragSvc)
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, kst)
		case "web_search":
			ws, err := websearch.New()
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, ws)
		case "web_fetch":
			wf, err := webfetch.New()
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, wf)
		default:
			return nil, errorer.New(errorer.ErrToolUnknown)
		}
	}
	return tools, nil
}
