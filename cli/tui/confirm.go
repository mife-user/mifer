package tui

import (
	"fmt"
	"mifer/cli/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// toolConfirmMsg SSE tool_confirm 事件到达
type toolConfirmMsg struct {
	callID string
	name   string
	args   string
}

// confirmOption 实现 bubbles/list.Item 接口
type confirmOption struct {
	label  string
	action string // "accept" | "refuse" | "allow"
}

func (o confirmOption) Title() string       { return o.label }
func (o confirmOption) Description() string { return "" }
func (o confirmOption) FilterValue() string { return o.label }

// handleToolConfirm 处理 tool_confirm 事件 → 显示确认选择列表
//
// 如果已有确认在进行中，将新请求追加到队列，避免覆盖。
func (m *Model) handleToolConfirm(msg toolConfirmMsg) (tea.Model, tea.Cmd) {
	if m.confirmingTool {
		// 已有确认在进行，追加到队列
		m.confirmQueue = append(m.confirmQueue, msg)
		info := fmt.Sprintf("工具确认排队: %s (队列 %d)", msg.name, len(m.confirmQueue))
		m.messages = append(m.messages, message{
			role:    "system",
			content: info,
		})
		m.needsAutoScroll = true
		return m, m.streamCmd()
	}

	info := "工具确认请求: " + msg.name
	if msg.args != "" {
		info += "\n参数: " + msg.args
	}
	m.messages = append(m.messages, message{
		role:    "system",
		content: info,
	})
	m.needsAutoScroll = true

	items := []list.Item{
		confirmOption{label: "接受", action: "accept"},
		confirmOption{label: "拒绝", action: "refuse"},
		confirmOption{label: "始终允许", action: "allow"},
	}
	m.confirmingTool = true
	m.confirmCallID = msg.callID
	m.confirmToolName = msg.name
	m.confirmToolArgs = msg.args
	m.confirmList.ResetSelected()
	return m, tea.Batch(m.confirmList.SetItems(items), listenStreamCmd(m.streamCh))
}

// handleConfirmSelect 处理确认选择列表中的 Enter 键
func (m *Model) handleConfirmSelect() (tea.Model, tea.Cmd) {
	action := "refuse"
	if item := m.confirmList.SelectedItem(); item != nil {
		action = item.(confirmOption).action
	}
	return m.resolveCurrentConfirm(action)
}

// resolveCurrentConfirm 发送当前确认决定，并检查队列是否有下一个待确认项
func (m *Model) resolveCurrentConfirm(action string) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{
		confirmCmd(m.client, m.confirmCallID, action),
		m.streamCmd(),
	}

	// 检查队列中是否还有待确认的工具
	if len(m.confirmQueue) > 0 {
		next := m.confirmQueue[0]
		m.confirmQueue = m.confirmQueue[1:]

		m.confirmCallID = next.callID
		m.confirmToolName = next.name
		m.confirmToolArgs = next.args

		info := "工具确认请求: " + next.name
		if next.args != "" {
			info += "\n参数: " + next.args
		}
		if len(m.confirmQueue) > 0 {
			info += fmt.Sprintf("\n(队列剩余 %d)", len(m.confirmQueue))
		}
		m.messages = append(m.messages, message{
			role:    "system",
			content: info,
		})
		m.needsAutoScroll = true

		m.confirmList.ResetSelected()
		cmds = append(cmds, m.confirmList.SetItems([]list.Item{
			confirmOption{label: "接受", action: "accept"},
			confirmOption{label: "拒绝", action: "refuse"},
			confirmOption{label: "始终允许", action: "allow"},
		}))
		return m, tea.Batch(cmds...)
	}

	// 队列为空，退出确认模式
	m.confirmingTool = false
	m.confirmCallID = ""
	m.confirmToolName = ""
	m.confirmToolArgs = ""
	return m, tea.Batch(cmds...)
}

// confirmCmd 异步发送工具确认决定到服务端
func confirmCmd(client *client.Client, callID, action string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Confirm.Confirm(callID, action); err != nil {
			return systemMsg{err: err}
		}
		return nil
	}
}
