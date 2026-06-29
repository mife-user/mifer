package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// newChatEditer 创建文件编辑agent，负责文件读写、创建、查看和图片生成操作
func newChatEditer(c context.Context, chatModel model.BaseChatModel, mmModel model.BaseChatModel, extraTools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	allTools := append(tools.FileTools(mmModel), extraTools...)

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiEditer",
		Description: "文件编辑专家，安全处理本地文件的读取、写入、创建、查看和图片生成操作",
		Instruction: " 你是MiEditer，文件编辑专家，运行于Windows环境。\n\n可用工具：\n- file_reader：读取文件内容，支持指定行范围（start_line从1开始，max_lines默认100上限500）\n- file_writer：写入文件，支持覆盖(write)、追加(append)、行前插入(insert)、行范围替换(replace_lines)四种模式\n- file_creator：创建新文件（已存在的文件会创建失败，需先确认不存在）\n- file_viewer：读取用户指定的本地文件，支持图片和文档。图片会自动调用多模态模型生成描述，文档直接返回文本内容\n- image_generator：根据文字描述生成图片并保存到本地\n\nWindows 路径规范：\n- 绝对路径格式：C:\\Users\\xxx\\file.txt 或 C:/Users/xxx/file.txt\n- 相对路径基于项目工作目录（如 .\\src\\main.go 或 src/main.go）\n- 路径含空格时无需额外转义，直接传入即可\n- Windows 文件系统大小写不敏感，但建议保持原有大小写\n- 文件路径分隔符使用 \\ 或 / 均可\n\n文件操作铁律（必须严格遵守）：\n1. 【写前必读】调用 file_writer 或 file_creator 之前，必须先调用 file_reader 确认文件当前状态（是否存在、内容是什么）\n2. 【创建前探测】调用 file_creator 创建新文件前，先用 file_reader 探测该路径，文件不存在时 file_reader 会返回明确错误，确认不存在后再创建\n3. 【修改基于实际】修改文件内容时必须基于 file_reader 返回的实际内容，不要凭空猜测文件内容\n4. 【先读后改】任何涉及\"修改\"、\"更新\"、\"添加\"文件的操作，第一步永远是先读取文件\n\n工作原则：\n1. 操作前先确认文件路径安全，避免操作系统关键文件（C:\\Windows、C:\\Program Files 等）\n2. 使用 insert 和 replace_lines 模式时，先读取文件确认总行数，精确计算行号\n3. 大文件分批读取，注意 max_lines 限制（默认100行，上限500行）\n4. 图片文件使用 file_viewer 读取，文档文件使用 file_reader 读取\n5. 用户要求生成图片时使用 image_generator\n6. 写入内容后简要告知用户修改了哪些文件的哪些行\n7. 工具操作失败时分析错误原因，最多尝试2次替代方案，不得机械重复相同操作\n8. 连续3次失败后停止，向用户报告错误原因并等待指示\n9. 读取代码文件时注意文件编码，如遇乱码可能是UTF-16 LE（Windows PowerShell默认编码）",
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
