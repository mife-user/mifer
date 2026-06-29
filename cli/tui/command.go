package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"mifer/cli/client"
	"mifer/pkg/conf"

	tea "github.com/charmbracelet/bubbletea"
)

// initPrefix 是 /init 命令发送给 AI 的项目初始化提示词。
// AI 会先探索项目结构并完整阅读源码，再生成 .mifer/MIFER.md 项目级提示词文件。
const initPrefix = `你是一个项目分析专家。请按以下步骤完成任务：

1. **探索结构**：先用工具列出项目根目录和各级子目录的内容，了解项目的目录组织方式，识别出：
   - 配置文件（构建系统、包管理、linter 等）
   - 源代码目录（如 src/、lib/、app/ 等，取决于具体语言和框架）
   - 已有的文档文件（README、CLAUDE.md 等）
   不要假设项目的语言或框架，一切从实际目录结构出发。

2. **阅读源码**：根据上一步探索的结果，使用 file_viewer 工具分批次阅读所有核心源文件和配置文件，确保：
   - 覆盖项目的所有模块和层级
   - 理解各模块之间的依赖和调用关系
   - 掌握项目的代码风格和命名约定

3. **阅读已有文档**：如果项目根目录存在 CLAUDE.md、README.md 或类似文档，阅读它们以补充理解。

4. **生成 MIFER.md**：使用 file_writer 工具在 .mifer/MIFER.md 创建项目级 AI 提示词文件，内容应包含：
   - 项目概述与用途
   - 技术栈与关键依赖
   - 目录结构与分层架构
   - 构建、运行、测试命令
   - 代码约定与命名风格
   - 关键设计模式与架构决策
   - 新增功能的开发指南

要求：内容简洁实用，使用中文编写，格式参考项目根目录下的 CLAUDE.md。完成后告知用户生成结果。`

// clearCmd 异步请求服务端生成新会话ID并切换
func clearCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		newID, err := client.Clear.Clear()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "已创建并切换到新会话: " + newID}
	}
}

// promptGetCmd 异步获取当前系统提示词
func promptGetCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		prompt, err := client.Prompt.Get()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "当前系统提示词:\n" + prompt}
	}
}

// promptSetCmd 异步设置自定义系统提示词
func promptSetCmd(client *client.Client, text string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Prompt.Set(text); err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "已设置系统提示词"}
	}
}

// promptResetCmd 异步重置为默认系统提示词
func promptResetCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.Prompt.Reset(); err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "已重置为默认系统提示词"}
	}
}

// skillListCmd 异步获取技能列表
func skillListCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		result, err := client.Skill.List()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: result}
	}
}

// mcpStatusCmd 异步获取 MCP Server 状态
func mcpStatusCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		result, err := client.MCP.Status()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: result}
	}
}

// agentsCmd 异步获取 Agent 列表
func agentsCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		result, err := client.Agents.List()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: result}
	}
}

// reloadCmd 异步请求服务端重载配置
func reloadCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		result, err := client.Reload.Reload()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: result}
	}
}

// checkBackendStatusCmd 启动时查询后端就绪状态，未配置 api_key 时在 TUI 显示警告
func checkBackendStatusCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		status, err := client.Status.Query()
		if err != nil {
			// 查询失败不阻塞启动（可能服务尚未就绪），静默忽略
			return backendStatusMsg{ready: false, warnings: nil}
		}
		return backendStatusMsg{ready: status.Ready, warnings: status.Warnings}
	}
}

// configFilePath 根据环境返回配置文件的绝对路径。
// dev → <Workdir>/config/dev.yaml
// prod → <CfgPath>/config/prod.yaml
func configFilePath(cfg *conf.Config) string {
	if cfg.Env == "dev" {
		return filepath.Join(cfg.Path.Workdir, "config", "dev.yaml")
	}
	return filepath.Join(cfg.Path.CfgPath, "config", "prod.yaml")
}

// resolveEditor 解析外部编辑器命令，返回可执行文件名及参数。
// 优先级：配置 cli.tui.editor → $VISUAL → $EDITOR → 平台默认
func resolveEditor(cfg *conf.Config) (string, []string) {
	var raw string
	if cfg.Cli.Tui.Editor != "" {
		raw = cfg.Cli.Tui.Editor
	} else if v := os.Getenv("VISUAL"); v != "" {
		raw = v
	} else if v := os.Getenv("EDITOR"); v != "" {
		raw = v
	}
	if raw != "" {
		parts := strings.Fields(raw)
		if len(parts) == 1 {
			return parts[0], nil
		}
		return parts[0], parts[1:]
	}
	if runtime.GOOS == "windows" {
		return "notepad", nil
	}
	return "vi", nil
}

// configCmd 调出外部编辑器编辑配置文件，关闭后自动重载配置。
func configCmd(client *client.Client) tea.Cmd {
	cfg := conf.GetConfig()
	cfgPath := configFilePath(cfg)

	// 校验配置文件存在
	info, err := os.Stat(cfgPath)
	if err != nil {
		return func() tea.Msg {
			return systemMsg{err: fmt.Errorf("配置文件不存在: %s", cfgPath)}
		}
	}
	if info.IsDir() {
		return func() tea.Msg {
			return systemMsg{err: fmt.Errorf("配置路径是目录而非文件: %s", cfgPath)}
		}
	}

	// 解析编辑器
	editorName, editorArgs := resolveEditor(cfg)
	cmd := exec.Command(editorName, append(editorArgs, cfgPath)...)

	// tea.Sequence: 先挂起 TUI 运行编辑器 → 显示结果 → 再执行 reload
	return tea.Sequence(
		tea.ExecProcess(cmd, func(execErr error) tea.Msg {
			if execErr != nil {
				return systemMsg{
					content: fmt.Sprintf("编辑器退出异常: %v\n正在重载配置...", execErr),
				}
			}
			return systemMsg{content: "编辑器已关闭"}
		}),
		reloadCmd(client),
	)
}

// compactCmd 异步请求服务端执行上下文压缩
func compactCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		result, err := client.Compact.Compact()
		if err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: result}
	}
}
