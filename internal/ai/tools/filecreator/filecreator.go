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

// New 创建 file_creator 工具，可选 baseDir 参数限制写入目录
func New(baseDir ...string) (tool.InvokableTool, error) {
	restrictDir := ""
	if len(baseDir) > 0 {
		restrictDir = baseDir[0]
	}
	return utils.InferTool("file_creator", "创建新文件，若文件已存在则返回错误。支持可选的初始内容写入，含路径安全校验。", func(ctx context.Context, input FileCreatorInput) (FileCreatorOutput, error) {
		return createFile(ctx, input, restrictDir)
	})
}

func createFile(_ context.Context, input FileCreatorInput, baseDir string) (FileCreatorOutput, error) {
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

	// 路径限制校验：仅允许在 baseDir 目录下创建文件
	if baseDir != "" {
		cleanBase, err := filepath.Abs(filepath.Clean(baseDir))
		if err != nil {
			return FileCreatorOutput{Error: "基准目录解析失败: " + err.Error()}, nil
		}
		baseSlash := filepath.ToSlash(cleanBase) + "/"
		pathSlash := filepath.ToSlash(absPath)
		if !strings.HasPrefix(pathSlash, baseSlash) || pathSlash == filepath.ToSlash(cleanBase) {
			return FileCreatorOutput{Error: "路径限制：仅允许在 " + cleanBase + " 目录下创建文件"}, nil
		}
	}

	dir := filepath.Dir(absPath)
	// 路径限制下，确保父目录也在限制范围内
	if baseDir != "" {
		cleanBase, _ := filepath.Abs(filepath.Clean(baseDir))
		baseSlash := filepath.ToSlash(cleanBase) + "/"
		if !strings.HasPrefix(filepath.ToSlash(dir)+"/", baseSlash) {
			return FileCreatorOutput{Error: "路径限制：不允许在限制目录外创建父目录"}, nil
		}
	}
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
