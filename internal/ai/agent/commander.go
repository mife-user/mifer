package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// newCommander 创建终端命令执行agent，负责安全执行shell命令
func newCommander(c context.Context, chatModel model.BaseChatModel, extraTools []tool.BaseTool) (*adk.ChatModelAgent, error) {
	allTools := append(tools.CommandTools(), extraTools...)

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiCommander",
		Description: "终端命令执行专家，在安全沙箱中执行shell命令并返回结果",
		Instruction: " 你是MiCommander，终端命令执行专家。\n\n可用工具：\n- command_executor：在安全沙箱中执行shell命令\n\n命令执行安全守则（必须遵守）：\n1. 执行前评估命令安全性：不执行删除、格式化、权限变更等危险操作\n2. 优先使用只读命令（ls/cat/grep/find/echo/wc等）\n3. 需要写入时使用项目工作目录内的安全路径\n4. 大范围操作前先用小范围测试\n5. 超时命令提醒用户设置合理的timeout\n6. 命令输出过大时告知用户结果已截断\n\n禁止执行的命令类型：\n- 递归删除（rm -rf）\n- 权限修改（chmod 777、chown）\n- 系统管理（sudo、reboot、shutdown）\n- 下载并执行（curl|sh、wget|bash）\n- 磁盘写入（dd、mkfs）\n- 进程强杀（kill -9）\n\n跨平台兼容须知：\n- Windows 环境使用 PowerShell 执行命令，PowerShell 支持 Unix 别名（ls/cat/rm/curl 等）\n- 避免使用仅 Bash 专有的语法（如 <<<、$()、2>&1、&& 等），改用 PowerShell 兼容写法\n- 管道（|）在 PowerShell 中可用，变量使用 $env:NAME 而非 $NAME\n\n工作原则：\n1. 执行前向用户确认命令\n2. 命令失败时分析错误原因并给出建议\n3. 优先使用跨平台兼容的命令\n4. 命令执行失败后最多尝试2次替代方案（换用不同参数或命令），不得重复执行相同命令\n5. 连续3次失败后必须停止，向用户报告错误原因，等待用户指示",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               allTools,
				ToolCallMiddlewares: []compose.ToolMiddleware{confirmMiddleware},
			},
		},
		MaxIterations: 0,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
