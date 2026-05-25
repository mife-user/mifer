package imagegenerator

import (
	"context"
	"encoding/base64"
	"fmt"
	"mifer/pkg/conf"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// ImageGeneratorInput 图片生成工具输入
type ImageGeneratorInput struct {
	Prompt     string `json:"prompt" jsonschema:"required,description=图片描述文本"`
	OutputPath string `json:"output_path" jsonschema:"description=输出文件路径（可选），默认保存到工作目录下以时间戳命名的文件"`
	Size       string `json:"size" jsonschema:"description=图片尺寸（可选），如 1024x1024"`
}

// ImageGeneratorOutput 图片生成工具输出
type ImageGeneratorOutput struct {
	ImagePath string `json:"image_path"`
	Error     string `json:"error,omitempty"`
}

// New 创建 image_generator 工具，mmModel 用于图片生成
func New(mmModel model.BaseChatModel) (tool.InvokableTool, error) {
	return utils.InferTool("image_generator",
		"根据文字描述生成图片。调用多模态模型生成图片并保存到本地，返回图片文件路径。",
		func(ctx context.Context, input ImageGeneratorInput) (ImageGeneratorOutput, error) {
			return generateImage(ctx, input, mmModel)
		})
}

func generateImage(ctx context.Context, input ImageGeneratorInput, mmModel model.BaseChatModel) (ImageGeneratorOutput, error) {
	if input.Prompt == "" {
		return ImageGeneratorOutput{Error: "图片描述不能为空"}, nil
	}

	if mmModel == nil {
		return ImageGeneratorOutput{Error: "多模态模型未配置，请在配置文件中设置 multi_modal 后端"}, nil
	}

	// 构建生成请求
	genPrompt := input.Prompt
	if input.Size != "" {
		genPrompt = fmt.Sprintf("%s\n输出尺寸: %s", input.Prompt, input.Size)
	}

	msg := &schema.Message{
		Role:    schema.User,
		Content: genPrompt,
	}

	resp, err := mmModel.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return ImageGeneratorOutput{Error: "图片生成失败: " + err.Error()}, nil
	}

	// 从响应中提取图片数据
	imgData, mimeType, err := extractImage(resp)
	if err != nil {
		return ImageGeneratorOutput{Error: err.Error()}, nil
	}

	// 确定输出路径
	outputPath := input.OutputPath
	if outputPath == "" {
		ext := ".png"
		switch mimeType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		}
		workdir := conf.GetConfig().Path.Workdir
		if workdir == "" {
			workdir = "."
		}
		outputPath = filepath.Join(workdir, fmt.Sprintf("generated_%s%s",
			time.Now().Format("20060102_150405"), ext))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return ImageGeneratorOutput{Error: "创建输出目录失败: " + err.Error()}, nil
	}

	if err := os.WriteFile(outputPath, imgData, 0644); err != nil {
		return ImageGeneratorOutput{Error: "保存图片失败: " + err.Error()}, nil
	}

	return ImageGeneratorOutput{ImagePath: outputPath}, nil
}

func extractImage(msg *schema.Message) ([]byte, string, error) {
	for _, part := range msg.AssistantGenMultiContent {
		if part.Type == schema.ChatMessagePartTypeImageURL && part.Image != nil {
			if part.Image.Base64Data != nil && *part.Image.Base64Data != "" {
				data, err := base64.StdEncoding.DecodeString(*part.Image.Base64Data)
				if err != nil {
					return nil, "", fmt.Errorf("解码图片数据失败: %w", err)
				}
				return data, part.Image.MIMEType, nil
			}
		}
	}

	return nil, "", fmt.Errorf("模型未返回图片数据，请尝试使用支持图片生成的模型（如 gemini-2.5-flash）")
}
