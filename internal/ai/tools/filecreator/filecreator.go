package filecreator

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// FileCreatorInput 文件创建工具输入参数
type FileCreatorInput struct {
	FilePath string `json:"file_path" jsonschema:"required,description=要创建的文件路径（绝对路径或相对路径）"`
	Content  string `json:"content" jsonschema:"description=文件的初始内容，可选"`
}

// FileCreatorOutput 文件创建工具输出结果
type FileCreatorOutput struct {
	Success      bool   `json:"success"`
	FilePath     string `json:"file_path"`
	BytesWritten int    `json:"bytes_written"`
	IsNew        bool   `json:"is_new"`
	Error        string `json:"error,omitempty"`
}

func New() (tool.InvokableTool, error) {
	return utils.InferTool("file_creator", "创建新文件，若文件已存在则返回错误。支持可选的初始内容写入，含路径安全校验。", createFile)
}

func createFile(_ context.Context, input FileCreatorInput) (FileCreatorOutput, error) {
	if input.FilePath == "" {
		return FileCreatorOutput{Error: "文件路径不能为空"}, nil
	}

	absPath, err := filepath.Abs(filepath.Clean(input.FilePath))
	if err != nil {
		return FileCreatorOutput{Error: "路径解析失败: " + err.Error()}, nil
	}
	if strings.Contains(filepath.ToSlash(absPath), "..") {
		absPath, _ = filepath.Abs(filepath.Clean(strings.ReplaceAll(input.FilePath, "..", "")))
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return FileCreatorOutput{Error: "创建目录失败: " + err.Error()}, nil
	}

	// O_EXCL 确保文件不存在时才创建，防止覆盖已有文件
	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return FileCreatorOutput{Error: "文件已存在: " + absPath}, nil
		}
		return FileCreatorOutput{Error: "创建文件失败: " + err.Error()}, nil
	}
	defer f.Close()

	n := 0
	if input.Content != "" {
		n, err = f.WriteString(input.Content)
		if err != nil {
			return FileCreatorOutput{Error: "写入内容失败: " + err.Error()}, nil
		}
	}

	return FileCreatorOutput{
		Success:      true,
		FilePath:     absPath,
		BytesWritten: n,
		IsNew:        true,
	}, nil
}
