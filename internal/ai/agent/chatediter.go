package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// newChatEditer 创建文件编辑agent，负责文件读写、创建、查看和图片生成操作
func newChatEditer(c context.Context, chatModel model.BaseChatModel, mmModel model.BaseChatModel, bus *tools.ConfirmBus) (*adk.ChatModelAgent, error) {
	cfg := &adk.ChatModelAgentConfig{
		Name:        "MiEditer",
		Description: "文件编辑专家，安全处理本地文件的读取、写入、创建、查看和图片生成操作",
		Instruction: " 你是MiEditer，文件编辑专家。\n\n可用工具：\n- file_reader：读取文件内容，支持指定行范围\n- file_writer：写入文件，支持覆盖(write)、追加(append)、行前插入(insert)、行范围替换(replace_lines)四种模式\n- file_creator：创建新文件（已存在的文件会创建失败）\n- file_viewer：读取用户指定的本地文件，支持图片和文档。图片会自动调用多模态模型生成描述，文档直接返回文本内容\n- image_generator：根据文字描述生成图片并保存到本地\n\n工作原则：\n1. 操作前先确认文件路径安全，避免操作系统关键文件\n2. 写入前先读取目标区域确认内容正确\n3. 使用insert和replace_lines时精确计算行号\n4. 大文件操作时分批读取（注意max_lines限制）\n5. 创建文件前确认文件不存在，避免覆盖已有文件\n6. 当用户提到图片文件路径时，使用 file_viewer 读取\n7. 当用户要求生成图片时，使用 image_generator 生成",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools.FileTools(mmModel),
			},
		},
		MaxIterations: 0,
	}
	if mw := makeConfirmMiddleware(bus); mw != nil {
		cfg.ToolsConfig.ToolCallMiddlewares = []compose.ToolMiddleware{{Invokable: mw}}
	}
	agent, err := adk.NewChatModelAgent(c, cfg)
	if err != nil {
		return nil, err
	}
	return agent, nil
}
