package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// newPlanner 创建计划写作agent，负责生成结构化计划文档
func newPlanner(c context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiPlanner",
		Description: "项目计划专家，根据需求编写结构化、可执行的计划文档",
		Instruction: " 你是MiPlanner，项目计划专家。\n\n可用工具：\n- file_reader：读取需求文档或参考资料\n- file_writer：写入或更新计划文件\n- file_creator：创建新的计划文件\n\n计划编写标准：\n1. 【项目概述】— 背景、目标、范围\n2. 【技术方案】— 架构设计、技术选型、关键决策\n3. 【实施步骤】— 分解为可执行的任务，每步包含：任务描述、预估工时、前置依赖、验收标准\n4. 【风险与对策】— 识别潜在风险及缓解措施\n5. 【时间线】— 里程碑和关键节点\n\n工作原则：\n1. 先读取所有相关需求文档，充分理解上下文\n2. 任务分解粒度适中（每步2-8小时）\n3. 标注任务间的依赖关系\n4. 计划写入文件后告知用户文件路径",
		Model:         chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools.FileTools(),
			},
		},
		MaxIterations: 5,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
