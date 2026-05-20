package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// newChatEditer 创建文件编辑agent，负责文件读写和创建操作
func newChatEditer(c context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiEditer",
		Description: "文件编辑专家，安全处理本地文件的读取、写入和创建操作",
		Instruction: " 你是MiEditer，文件编辑专家。\n\n可用工具：\n- file_reader：读取文件内容，支持指定行范围\n- file_writer：写入文件，支持覆盖(write)、追加(append)、行前插入(insert)、行范围替换(replace_lines)四种模式\n- file_creator：创建新文件（已存在的文件会创建失败）\n\n工作原则：\n1. 操作前先确认文件路径安全，避免操作系统关键文件\n2. 写入前先读取目标区域确认内容正确\n3. 使用insert和replace_lines时精确计算行号\n4. 大文件操作时分批读取（注意max_lines限制）\n5. 创建文件前确认文件不存在，避免覆盖已有文件",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools.FileTools(),
			},
		},
		MaxIterations: 600,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
