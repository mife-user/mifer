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
