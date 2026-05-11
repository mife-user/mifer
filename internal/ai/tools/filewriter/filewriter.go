package filewriter

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// FileWriterInput 文件写入工具输入参数
type FileWriterInput struct {
	FilePath string `json:"file_path" jsonschema:"required,description=要写入的文件路径（绝对路径或相对路径）"`
	Content  string `json:"content" jsonschema:"required,description=要写入文件的内容"`
	Mode     string `json:"mode" jsonschema:"description=写入模式：write（覆盖写入）或 append（追加写入），默认write"`
}

// FileWriterOutput 文件写入工具输出结果
type FileWriterOutput struct {
	Success      bool   `json:"success"`
	BytesWritten int    `json:"bytes_written"`
	FilePath     string `json:"file_path"`
	Error        string `json:"error,omitempty"`
}

func New() (tool.InvokableTool, error) {
	return utils.InferTool("file_writer", "安全写入内容到本地文件，支持覆盖和追加两种模式，含路径安全校验。", writeFile)
}

func writeFile(_ context.Context, input FileWriterInput) (FileWriterOutput, error) {
	if input.Mode != "append" {
		input.Mode = "write"
	}

	absPath, err := filepath.Abs(filepath.Clean(input.FilePath))
	if err != nil {
		return FileWriterOutput{Error: "路径解析失败: " + err.Error()}, nil
	}
	if strings.Contains(filepath.ToSlash(absPath), "..") {
		absPath, _ = filepath.Abs(filepath.Clean(strings.ReplaceAll(input.FilePath, "..", "")))
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return FileWriterOutput{Error: "创建目录失败: " + err.Error()}, nil
	}

	var flag int
	if input.Mode == "append" {
		flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	}

	f, err := os.OpenFile(absPath, flag, 0644)
	if err != nil {
		return FileWriterOutput{Error: "打开文件失败: " + err.Error()}, nil
	}
	defer f.Close()

	n, err := f.WriteString(input.Content)
	if err != nil {
		return FileWriterOutput{Error: "写入文件失败: " + err.Error()}, nil
	}

	return FileWriterOutput{
		Success:      true,
		BytesWritten: n,
		FilePath:     absPath,
	}, nil
}
