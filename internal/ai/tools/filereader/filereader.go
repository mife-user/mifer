package filereader

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// FileReaderInput 文件读取工具输入参数
type FileReaderInput struct {
	FilePath string `json:"file_path" jsonschema:"required,description=要读取的文件路径（绝对路径或相对路径）"`
	MaxLines int    `json:"max_lines" jsonschema:"description=最大读取行数，默认100，上限500"`
}

// FileReaderOutput 文件读取工具输出结果
type FileReaderOutput struct {
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}

func New() (tool.InvokableTool, error) {
	return utils.InferTool("file_reader", "安全读取本地文本文件内容，支持行数限制和路径安全校验。", readFile)
}

func readFile(_ context.Context, input FileReaderInput) (FileReaderOutput, error) {
	if input.MaxLines <= 0 {
		input.MaxLines = 100
	}
	if input.MaxLines > 500 {
		input.MaxLines = 500
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

	var lines []string
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() && count < input.MaxLines {
		lines = append(lines, scanner.Text())
		count++
	}

	truncated := scanner.Scan()

	return FileReaderOutput{
		Content:   strings.Join(lines, "\n"),
		LineCount: count,
		Truncated: truncated,
	}, nil
}
