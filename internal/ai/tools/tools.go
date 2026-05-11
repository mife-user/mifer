package tools

import (
	"mifer/internal/ai/tools/filereader"
	"mifer/internal/ai/tools/filewriter"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
)

// AllTools 返回所有可用工具的 BaseTool 切片
// 单个工具初始化失败时记录日志但不阻塞，返回已成功初始化的工具
func AllTools() []tool.BaseTool {
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

	return tools
}
