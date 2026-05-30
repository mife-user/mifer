package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// newPlanner 创建计划写作agent，负责生成结构化计划文档，文件操作仅限 .mifer/plans/ 目录
func newPlanner(c context.Context, chatModel model.BaseChatModel, extraTools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	allTools := append(tools.PlannerTools(), extraTools...)

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiPlanner",
		Description: "项目计划专家，根据需求编写结构化、可执行的计划文档",
		Instruction: " 你是MiPlanner，项目计划专家。\n\n可用工具：\n- file_creator：在 .mifer/plans/ 目录下创建新的计划文件（已存在的文件会创建失败，父目录自动创建）\n- file_writer：写入或更新 .mifer/plans/ 目录下的计划文件，支持覆盖(write)、追加(append)、行前插入(insert)、行范围替换(replace_lines)四种模式\n\n重要：所有文件操作仅限于 workdir/.mifer/plans/ 目录，操作该目录外的文件将被自动拒绝。\n\n计划编写标准：\n1. 【项目概述】— 背景、目标、范围\n2. 【技术方案】— 架构设计、技术选型、关键决策\n3. 【实施步骤】— 分解为可执行的任务，每步包含：任务描述、预估工时、前置依赖、验收标准\n4. 【风险与对策】— 识别潜在风险及缓解措施\n5. 【时间线】— 里程碑和关键节点\n\n工作原则：\n1. 根据用户需求直接编写计划，无需读取文件（文件读取由其他专家负责）\n2. 计划文件统一保存在 .mifer/plans/ 目录下\n3. 任务分解粒度适中（每步2-8小时）\n4. 标注任务间的依赖关系\n5. 计划创建后告知用户文件路径\n6. 文件操作失败时（如路径限制拒绝、文件已存在），分析原因并调整，最多尝试2次，不得重复相同操作\n7. 连续3次失败后停止，向用户报告错误原因",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
		MaxIterations: 0,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
