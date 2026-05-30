package tui

import (
	"mifer/cli/client"

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
