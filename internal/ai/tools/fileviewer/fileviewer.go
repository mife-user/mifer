package fileviewer

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	mimetype_detect "github.com/gabriel-vasile/mimetype"
)

// FileViewerInput 文件查看工具输入
type FileViewerInput struct {
	FilePaths []string `json:"file_paths" jsonschema:"required,description=要读取的文件路径列表"`
}

// FileResult 单个文件的读取结果
type FileResult struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	MIMEType string `json:"mime_type"`
	IsImage  bool   `json:"is_image"`
	Error    string `json:"error,omitempty"`
}

// FileViewerOutput 文件查看工具输出
type FileViewerOutput struct {
	Results []FileResult `json:"results"`
	Error   string       `json:"error,omitempty"`
}

// New 创建 file_viewer 工具，mmModel 用于图片描述（可为 nil，此时图片读取返回错误）
func New(mmModel model.BaseChatModel) (tool.InvokableTool, error) {
	return utils.InferTool("file_viewer",
		"读取用户指定的本地文件。对图片文件会调用多模态模型生成图片描述，对文档文件直接返回文本内容。支持批量读取。",
		func(ctx context.Context, input FileViewerInput) (FileViewerOutput, error) {
			return viewFiles(ctx, input, mmModel)
		})
}

func viewFiles(ctx context.Context, input FileViewerInput, mmModel model.BaseChatModel) (FileViewerOutput, error) {
	if len(input.FilePaths) == 0 {
		return FileViewerOutput{Error: "文件路径列表为空"}, nil
	}

	results := make([]FileResult, 0, len(input.FilePaths))

	for _, fp := range input.FilePaths {
		results = append(results, readOneFile(ctx, fp, mmModel))
	}

	return FileViewerOutput{Results: results}, nil
}

func readOneFile(ctx context.Context, fp string, mmModel model.BaseChatModel) FileResult {
	absPath, err := filepath.Abs(filepath.Clean(fp))
	if err != nil {
		return FileResult{FilePath: fp, Error: "路径解析失败: " + err.Error()}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileResult{FilePath: absPath, Error: "文件不存在"}
		}
		return FileResult{FilePath: absPath, Error: "读取文件失败: " + err.Error()}
	}

	mtype := mimetype_detect.Detect(data)
	mimeStr := mtype.String()

	// 图片文件：调多模态模型生成描述
	if strings.HasPrefix(mimeStr, "image/") {
		desc, err := describeImage(ctx, data, mimeStr, mmModel)
		if err != nil {
			return FileResult{FilePath: absPath, MIMEType: mimeStr, IsImage: true, Error: "图片描述失败: " + err.Error()}
		}
		return FileResult{
			FilePath: absPath,
			Content:  desc,
			MIMEType: mimeStr,
			IsImage:  true,
		}
	}

	// 文本类文件：直接返回内容
	if isTextMIME(mimeStr) {
		content := string(data)
		if len(content) > 100000 {
			content = content[:100000] + fmt.Sprintf("\n\n[... 内容已截断，原始长度 %d 字符]", len(content))
		}
		return FileResult{
			FilePath: absPath,
			Content:  content,
			MIMEType: mimeStr,
		}
	}

	// 其他二进制文件
	return FileResult{
		FilePath: absPath,
		MIMEType: mimeStr,
		Error:    fmt.Sprintf("无法读取二进制文件（MIME: %s），不支持此格式", mimeStr),
	}
}

func describeImage(ctx context.Context, data []byte, mimeType string, mmModel model.BaseChatModel) (string, error) {
	if mmModel == nil {
		return "", fmt.Errorf("多模态模型未配置，请在配置文件中设置 multi_modal 后端")
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	msg := &schema.Message{
		Role: schema.User,
		UserInputMultiContent: []schema.MessageInputPart{
			{Type: schema.ChatMessagePartTypeText, Text: "请详细描述这张图片里的内容。"},
			{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &b64,
					MIMEType:   mimeType,
				},
			}},
		},
	}

	resp, err := mmModel.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// isTextMIME 判断 MIME 类型是否为可读文本
func isTextMIME(m string) bool {
	if strings.HasPrefix(m, "text/") {
		return true
	}
	textTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-yaml",
		"application/x-httpd-php",
	}
	for _, t := range textTypes {
		if m == t {
			return true
		}
	}
	return false
}
