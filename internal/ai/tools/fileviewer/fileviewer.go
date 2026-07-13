package fileviewer

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	mimetype_detect "github.com/gabriel-vasile/mimetype"
)

// FileViewerInput 文件查看工具输入
type FileViewerInput struct {
	FilePaths []string `json:"file_paths" jsonschema:"required,description=图片文件路径列表，支持批量查看多张图片"`
}

// New 创建 file_viewer 工具（EnhancedInvokableTool）
// 读取图片文件并返回结构化图片数据供 LLM 原生识别。文本文件请使用 file_reader。
func New() (tool.EnhancedInvokableTool, error) {
	return utils.InferEnhancedTool("file_viewer",
		"读取图片文件并返回图片数据供AI直接识别。支持批量查看多张图片。注意：非图片文件（文本、代码等）请使用 file_reader 读取。",
		func(ctx context.Context, input FileViewerInput) (*schema.ToolResult, error) {
			return viewFiles(ctx, input)
		})
}

func viewFiles(ctx context.Context, input FileViewerInput) (*schema.ToolResult, error) {
	if len(input.FilePaths) == 0 {
		return &schema.ToolResult{Parts: []schema.ToolOutputPart{
			{Type: schema.ToolPartTypeText, Text: "文件路径列表为空"},
		}}, nil
	}

	var parts []schema.ToolOutputPart

	for _, fp := range input.FilePaths {
		part := readOneFile(ctx, fp)
		parts = append(parts, part)
	}

	return &schema.ToolResult{Parts: parts}, nil
}

func readOneFile(ctx context.Context, fp string) schema.ToolOutputPart {
	absPath, err := filepath.Abs(filepath.Clean(fp))
	if err != nil {
		return schema.ToolOutputPart{
			Type: schema.ToolPartTypeText,
			Text: fmt.Sprintf("文件 %s: 路径解析失败: %s", fp, err),
		}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return schema.ToolOutputPart{
				Type: schema.ToolPartTypeText,
				Text: fmt.Sprintf("文件 %s: 文件不存在", absPath),
			}
		}
		return schema.ToolOutputPart{
			Type: schema.ToolPartTypeText,
			Text: fmt.Sprintf("文件 %s: 读取失败: %s", absPath, err),
		}
	}

	mtype := mimetype_detect.Detect(data)
	mimeStr := mtype.String()

	// 图片文件：返回结构化图片数据
	if strings.HasPrefix(mimeStr, "image/") {
		b64 := base64.StdEncoding.EncodeToString(data)
		return schema.ToolOutputPart{
			Type: schema.ToolPartTypeImage,
			Image: &schema.ToolOutputImage{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &b64,
					MIMEType:   mimeStr,
				},
			},
		}
	}

	// 非图片文件：提示使用 file_reader
	return schema.ToolOutputPart{
		Type: schema.ToolPartTypeText,
		Text: fmt.Sprintf("文件 %s（%s）不是图片，无法查看。文本文件请使用 file_reader 读取", absPath, mimeStr),
	}
}
