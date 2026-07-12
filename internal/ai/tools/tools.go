package tools

import (
	"path/filepath"

	"mifer/internal/ai/rag"
	"mifer/internal/ai/tools/commandexecutor"
	"mifer/internal/ai/tools/filecreator"
	"mifer/internal/ai/tools/filereader"
	"mifer/internal/ai/tools/fileviewer"
	"mifer/internal/ai/tools/filewriter"
	"mifer/internal/ai/tools/imagegenerator"
	"mifer/internal/ai/tools/knowledgesearch"
	"mifer/internal/ai/tools/knowledgestore"
	"mifer/internal/ai/tools/paralleldispatch"
	qqtools "mifer/internal/ai/tools/qq"
	"mifer/internal/ai/tools/webfetch"
	"mifer/internal/ai/tools/websearch"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"mifer/pkg/skill"
	"mifer/qq"

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

// PlannerTools 返回计划编写专用工具（仅文件创建和写入，限制在 .mifer/plans 目录下）
func PlannerTools() []tool.BaseTool {
	var tools []tool.BaseTool
	plansDir := filepath.Join(conf.GetConfig().Path.Workdir, ".mifer", "plans")

	fc, err := filecreator.New(plansDir)
	if err != nil {
		logger.Error("创建 planner file_creator 工具失败", logger.C(err))
	} else {
		tools = append(tools, fc)
	}

	fw, err := filewriter.New(plansDir)
	if err != nil {
		logger.Error("创建 planner file_writer 工具失败", logger.C(err))
	} else {
		tools = append(tools, fw)
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

// ParallelDispatch 返回并行调度工具，agentHub 用于查找目标 Agent 实例
func ParallelDispatch(agentHub *skill.AgentHub) []tool.BaseTool {
	var tools []tool.BaseTool

	pd, err := paralleldispatch.New(agentHub)
	if err != nil {
		logger.Error("创建 parallel_dispatch 工具失败", logger.C(err))
	} else {
		tools = append(tools, pd)
	}

	return tools
}

// ReadOnlyTools 返回只读工具（用于计划 Agent），不可写入、不可执行命令。
func ReadOnlyTools(mmModel model.BaseChatModel, ragSvc rag.RAGService) []tool.BaseTool {
	var ts []tool.BaseTool

	fr, err := filereader.New()
	if err != nil {
		logger.Error("创建 file_reader 工具失败", logger.C(err))
	} else {
		ts = append(ts, fr)
	}

	fv, err := fileviewer.New(mmModel)
	if err != nil {
		logger.Error("创建 file_viewer 工具失败", logger.C(err))
	} else {
		ts = append(ts, fv)
	}

	for _, t := range WebTools() {
		ts = append(ts, t)
	}

	if ragSvc != nil {
		ks, err := knowledgesearch.New(ragSvc)
		if err != nil {
			logger.Error("创建 knowledge_search 工具失败", logger.C(err))
		} else {
			ts = append(ts, ks)
		}
	}

	return ts
}

func NewWithName(name []string, mmModel model.BaseChatModel, ragSvc rag.RAGService) ([]tool.BaseTool, error) {
	var tools []tool.BaseTool
	for _, n := range name {
		switch n {
		case "file_reader":
			fr, err := filereader.New()
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, fr)
		case "file_writer":
			fw, err := filewriter.New()
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, fw)
		case "file_creator":
			fc, err := filecreator.New()
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, fc)
		case "file_viewer":
			fv, err := fileviewer.New(mmModel)
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, fv)
		case "image_generator":
			ig, err := imagegenerator.New(mmModel)
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, ig)
		case "command_executor":
			ce, err := commandexecutor.New()
			if err != nil {
				logger.Error("创建工具失败", logger.S("tool", n), logger.C(err))
				return nil, err
			}
			tools = append(tools, ce)
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
