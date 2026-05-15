package tools

import (
	"mifer/internal/ai/tools/commandexecutor"
	"mifer/internal/ai/tools/filecreator"
	"mifer/internal/ai/tools/filereader"
	"mifer/internal/ai/tools/filewriter"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
)

// FileTools 返回文件操作相关工具（读取、写入、创建）
func FileTools() []tool.BaseTool {
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

	return tools
}

// CommandTools 返回命令执行相关工具
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

// AuditTools 返回安全审计相关工具（仅文件读取）
func AuditTools() []tool.BaseTool {
	var tools []tool.BaseTool

	fr, err := filereader.New()
	if err != nil {
		logger.Error("创建 file_reader 工具失败", logger.C(err))
	} else {
		tools = append(tools, fr)
	}

	return tools
}

// AllTools 返回所有可用工具的 BaseTool 切片（向后兼容）
func AllTools() []tool.BaseTool {
	var tools []tool.BaseTool
	tools = append(tools, FileTools()...)
	tools = append(tools, CommandTools()...)
	return tools
}
