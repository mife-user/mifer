package tui

// ============================================================================
// toolconfirm_args.go — 工具参数 DTO 与格式化函数
// ============================================================================
//
// 各工具参数字段定义，用于 JSON 反序列化与格式化展示。
// 从 toolconfirm.go 中拆分，独立管理参数展示逻辑。

import (
	"fmt"

	"mifer/pkg/exc"
)

// ============================================================================
// 工具参数 DTO
// ============================================================================

type cmdArgs struct {
	Command string `json:"command"`
}
type fileCreateArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}
type fileWriteArgs struct {
	FilePath  string `json:"file_path"`
	Content   string `json:"content"`
	Mode      string `json:"mode"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}
type fileReadArgs struct {
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	MaxLines  int    `json:"max_lines"`
}
type webSearchArgs struct {
	Query string `json:"query"`
}
type webFetchArgs struct {
	URL string `json:"url"`
}
type imageGenArgs struct {
	Prompt string `json:"prompt"`
	Output string `json:"output"`
}
type knowledgeSearchArgs struct {
	Query string `json:"query"`
}
type knowledgeStoreArgs struct {
	FilePath string `json:"file_path"`
}

// ============================================================================
// 工具函数
// ============================================================================

// formatConfirmArgs 根据工具类型格式化参数用于对话框展示。
func formatConfirmArgs(toolName, argsJSON string) string {
	switch toolName {
	case "command_executor":
		var a cmdArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Command != "" {
			return "   命令: " + a.Command
		}
	case "file_creator":
		var a fileCreateArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil {
			s := "   文件: " + a.FilePath
			if a.Content != "" {
				s += "\n   内容: " + a.Content
			}
			return s
		}
	case "file_writer":
		var a fileWriteArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil {
			s := fmt.Sprintf("   文件: %s\n   模式: %s", a.FilePath, a.Mode)
			if a.StartLine > 0 || a.EndLine > 0 {
				s += fmt.Sprintf(" (行 %d-%d)", a.StartLine, a.EndLine)
			}
			if a.Content != "" {
				s += "\n   内容: " + a.Content
			}
			return s
		}
	case "file_reader", "file_viewer":
		var a fileReadArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil {
			s := "   文件: " + a.FilePath
			if a.StartLine > 0 || a.MaxLines > 0 {
				s += fmt.Sprintf("\n   行范围: %d~%d", a.StartLine, a.StartLine+a.MaxLines)
			}
			return s
		}
	case "web_search":
		var a webSearchArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Query != "" {
			return "   搜索: " + a.Query
		}
	case "web_fetch":
		var a webFetchArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.URL != "" {
			return "   网址: " + a.URL
		}
	case "image_generator":
		var a imageGenArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Prompt != "" {
			s := "   提示词: " + a.Prompt
			if a.Output != "" {
				s += "\n   输出: " + a.Output
			}
			return s
		}
	case "knowledge_search":
		var a knowledgeSearchArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Query != "" {
			return "   检索: " + a.Query
		}
	case "knowledge_store":
		var a knowledgeStoreArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.FilePath != "" {
			return "   文件: " + a.FilePath
		}
	}
	// 通用降级
	var generic map[string]any
	if exc.ExcJSONToFile(argsJSON, &generic) == nil {
		c, _ := exc.ExcFileToJSON(generic)
		return "   参数: " + c
	}
	return ""
}

// parseCommandForAllowlist 从 command_executor 的参数中解析出命令字符串。
func parseCommandForAllowlist(argsJSON string) string {
	var a cmdArgs
	if exc.ExcJSONToFile(argsJSON, &a) == nil {
		return a.Command
	}
	return ""
}
