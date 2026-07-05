package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// newImage 创建图片生成的 Agent
func newImage(c context.Context, chatModel model.BaseChatModel, mmModel model.BaseChatModel, extraTools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	allTools := append(tools.Image(mmModel), extraTools...)

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiImager",
		Description: "进行图片查看与生成操作",
		Instruction: " 你是MiImager，图片生成专家，运行于Windows环境。\n\n可用工具：\n- image_generator：根据文字描述生成图片并保存到本地\n\nWindows 路径规范：\n- 绝对路径格式：C:\\Users\\xxx\\file.txt 或 C:/Users/xxx/file.txt\n- 相对路径基于项目工作目录（如 .\\src\\main.go 或 src/main.go）\n- 路径含空格时无需额外转义，直接传入即可\n- Windows 文件系统大小写不敏感，但建议保持原有大小写\n- 文件路径分隔符使用 \\ 或 / 均可\n\n工作原则：\n1. 操作前先确认文件路径安全，避免操作系统关键文件（C:\\Windows、C:\\Program Files 等）\n2. 用户要求生成图片时使用 image_generator\n3. 写入内容后简要告知用户修改了哪些文件的哪些行\n4. 工具操作失败时分析错误原因，最多尝试2次替代方案，不得机械重复相同操作\n5. 连续3次失败后停止，向用户报告错误原因并等待指示",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               allTools,
				ToolCallMiddlewares: []compose.ToolMiddleware{confirmMiddleware},
			},
		},
		MaxIterations: 100,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
