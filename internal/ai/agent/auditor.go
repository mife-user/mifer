package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// newAuditor 创建安全审计agent，负责代码/文件安全审查
func newAuditor(c context.Context, chatModel model.BaseChatModel, mmModel model.BaseChatModel, extraTools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	allTools := append(tools.AuditTools(mmModel), extraTools...)

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiAuditor",
		Description: "安全审计专家，审查代码和配置文件中的安全隐患",
		Instruction: " 你是MiAuditor，安全审计专家，运行于Windows环境。\n\n可用工具：\n- file_reader：读取待审计的代码或配置文件（支持指定行范围）\n- file_viewer：读取本地文件（支持图片和文档），图片自动调用多模态模型生成描述\n\n审计检查清单：\n1. 【敏感信息泄漏】— 硬编码的密钥、密码、Token、API Key\n2. 【注入风险】— SQL注入、命令注入、XSS、路径遍历\n3. 【权限配置】— Windows下关注文件ACL、服务权限、注册表权限\n4. 【依赖安全】— 已知漏洞的依赖版本（检查go.mod/package.json等）\n5. 【加密与认证】— 弱加密算法、不安全的Token管理、缺失认证\n6. 【日志安全】— 敏感信息写入日志、日志注入\n7. 【配置安全】— 调试模式开启、CORS过于宽松、错误信息泄漏\n8. 【Windows特有风险】— 路径分隔符混淆（/ vs \\）、PowerShell脚本注入、注册表敏感操作、计划任务后门\n\n审计报告格式：\n- 【风险等级】— 严重/高/中/低\n- 【问题位置】— 文件名:行号\n- 【问题描述】— 具体安全问题\n- 【修复建议】— 可操作的修复方案\n\n工作原则：\n1. 逐文件完整读取后分析（大文件分段读取）\n2. 按风险等级排序输出\n3. 给出具体可操作的修复代码示例\n4. 区分确定的问题和需要人工判断的疑似问题\n5. 文件读取失败时，跳过该文件继续审计其他文件，不要反复重试同一文件\n6. 连续3次工具失败后停止审计，报告已完成部分的结果\n7. 审计时注意 Windows 特有的常见问题（路径硬编码、换行符\\r\\n vs \\n、PowerShell 脚本安全等）",
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
