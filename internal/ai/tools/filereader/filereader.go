package filereader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// FileReaderInput 文件读取工具输入参数
type FileReaderInput struct {
	FilePath  string `json:"file_path" jsonschema:"required,description=要读取的文件路径（绝对路径或相对路径）"`
	StartLine int    `json:"start_line" jsonschema:"description=起始行号（1-based），默认从第1行开始读取"`
	MaxLines  int    `json:"max_lines" jsonschema:"description=最大读取行数，默认100，上限500"`
}

// FileReaderOutput 文件读取工具输出结果
type FileReaderOutput struct {
	Content    string `json:"content"`
	LineCount  int    `json:"line_count"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	TotalLines int    `json:"total_lines"`
	Truncated  bool   `json:"truncated"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

func New() (tool.InvokableTool, error) {
	return utils.InferTool("file_reader", "安全读取本地文本文件内容，支持指定起始行号、行数限制和路径安全校验。", readFile)
}

func readFile(_ context.Context, input FileReaderInput) (FileReaderOutput, error) {
	var messages []string

	startLine := input.StartLine
	if startLine <= 0 {
		startLine = 1
		messages = append(messages, "起始行号不能小于1，已自动调整为第1行")
	}

	maxLines := input.MaxLines
	if maxLines <= 0 {
		maxLines = 100
		messages = append(messages, "读取行数无效，已自动调整为默认100行")
	}
	if maxLines > 500 {
		maxLines = 500
		messages = append(messages, "读取行数超过上限500，已自动调整为500行")
	}

	absPath, err := filepath.Abs(filepath.Clean(input.FilePath))
	if err != nil {
		return FileReaderOutput{Error: "路径解析失败: " + err.Error()}, nil
	}
	if strings.Contains(filepath.ToSlash(absPath), "..") {
		// filepath.Clean 之后仍有 .. 说明路径不安全
		absPath, _ = filepath.Abs(filepath.Clean(strings.ReplaceAll(input.FilePath, "..", "")))
	}

	f, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileReaderOutput{Error: "文件不存在: " + absPath}, nil
		}
		return FileReaderOutput{Error: "打开文件失败: " + err.Error()}, nil
	}
	defer f.Close()

	// 单遍扫描：边跳行边读取，同时统计总行数
	scanner := bufio.NewScanner(f)
	var lines []string
	lineNum := 0       // 当前行号（1-based，0 表示尚未读到任何行）
	count := 0         // 实际读取的行数
	afterCount := 0    // 读取完毕后剩余行数（用于检测截断）

	for scanner.Scan() {
		lineNum++
		if lineNum < startLine {
			continue // 还没到起始行，跳过
		}
		if count < maxLines {
			lines = append(lines, scanner.Text())
			count++
		} else {
			afterCount++
		}
	}
	totalLines := lineNum

	// 空文件直接返回
	if totalLines == 0 {
		return FileReaderOutput{
			Content:    "",
			LineCount:  0,
			StartLine:  startLine,
			EndLine:    0,
			TotalLines: 0,
			Message:    strings.Join(messages, "; "),
		}, nil
	}

	// 起始行号超出文件总行数
	if startLine > totalLines {
		return FileReaderOutput{
			Error:      fmt.Sprintf("起始行号%d超出文件总行数%d", startLine, totalLines),
			StartLine:  startLine,
			TotalLines: totalLines,
		}, nil
	}

	endLine := startLine + count - 1

	// 检查是否因到达文件末尾导致实际读取行数少于请求
	if count < maxLines && endLine == totalLines {
		messages = append(messages,
			fmt.Sprintf("请求读取%d行，但起始行%d到文件末尾仅剩%d行，实际读取%d行",
				maxLines, startLine, totalLines-startLine+1, count))
	}

	truncated := afterCount > 0

	return FileReaderOutput{
		Content:    strings.Join(lines, "\n"),
		LineCount:  count,
		StartLine:  startLine,
		EndLine:    endLine,
		TotalLines: totalLines,
		Truncated:  truncated,
		Message:    strings.Join(messages, "; "),
	}, nil
}
