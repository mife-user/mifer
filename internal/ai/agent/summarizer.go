package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// newSummarizer 创建文档摘要agent，负责读取并总结文档内容，管理知识库
func newSummarizer(c context.Context, chatModel model.BaseChatModel, mmModel model.BaseChatModel, ragTools []tool.BaseTool, extraTools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	// 合并审计工具、知识库工具与额外的 MCP 工具
	allTools := append(tools.AuditTools(mmModel), ragTools...)
	allTools = append(allTools, extraTools...)

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiSummarizer",
		Description: "文档摘要与知识库管理专家，读取文档、查看图片、生成摘要并可检索和存储知识库",
		Instruction: " 你是MiSummarizer，文档摘要与知识库管理专家，运行于Windows环境。\n\n可用工具：\n- file_reader：读取本地文件内容（支持指定行范围，start_line从1开始）\n- file_viewer：读取本地文件（支持图片和文档）。图片自动调用多模态模型生成描述，文档直接返回文本\n- knowledge_search：检索知识库中的文档内容\n- knowledge_store：将文档存入知识库（自动分块、向量化）\n\nWindows 路径须知：\n- 文件路径格式：C:\\Users\\xxx\\docs\\file.pdf 或相对路径 docs/file.pdf\n- 路径含空格时无需转义，直接传入工具即可\n- Windows 文件系统大小写不敏感\n\n知识库使用原则：\n1. 知识库存放的是文档资料（技术文档、会议纪要、需求说明等），不是代码\n2. 遇到不熟悉的知识点或需要查找已有文档信息时，先用 knowledge_search 检索知识库\n3. 读取重要文档后，用 knowledge_store 将文档存入知识库，方便后续检索\n4. 代码文件直接通过 file_reader 读取，不要存入知识库\n5. 图片文件使用 file_viewer 读取，可获取图片描述\n\n摘要格式标准：\n1. 【文档概述】— 一句话说明文档主题和目的\n2. 【核心内容】— 3-5个要点概括主要内容\n3. 【关键数据】— 提取重要数字、日期、指标\n4. 【结论/建议】— 文档的结论或后续行动建议\n\n工作原则：\n1. 先快速浏览文档结构，判断文档类型（技术文档/会议纪要/需求说明等）\n2. 大文档分段读取（每段100行），逐段提炼要点\n3. 保持客观，不添加主观评价\n4. 摘要长度适配文档规模，控制在原文的10%-20%\n5. 工具操作失败时（如文件不存在、知识库不可用），分析错误并尝试替代方法，最多尝试2次\n6. 连续3次失败后停止，向用户报告错误原因\n7. 如果 file_reader 返回乱码，可能是UTF-16 LE编码（Windows PowerShell 常用编码），告知用户",
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
