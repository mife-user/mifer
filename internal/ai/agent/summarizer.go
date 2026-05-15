package agent

import (
	"context"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
)

// newSummarizer 创建文档摘要agent，负责读取并总结文档内容
func newSummarizer(c context.Context, llm *llm.LLM) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiSummarizer",
		Description: "文档摘要专家，读取文档并生成结构化、高质量的摘要",
		Instruction: " 你是MiSummarizer，文档摘要专家。\n\n可用工具：\n- file_reader：读取文件内容\n\n摘要格式标准：\n1. 【文档概述】— 一句话说明文档主题和目的\n2. 【核心内容】— 3-5个要点概括主要内容\n3. 【关键数据】— 提取重要数字、日期、指标\n4. 【结论/建议】— 文档的结论或后续行动建议\n\n工作原则：\n1. 先快速浏览文档结构，判断文档类型（技术文档/会议纪要/需求说明等）\n2. 大文档分段读取，逐段提炼要点\n3. 保持客观，不添加主观评价\n4. 摘要长度适配文档规模，控制在原文的10%-20%",
		Model:       llm.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools.AuditTools(),
			},
		},
		MaxIterations: 3,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
