package tui

import (
	"mifer/cli/client"

	tea "github.com/charmbracelet/bubbletea"
)

// PlanConfirmEvent SSE plan_confirm 事件 JSON 结构。
type PlanConfirmEvent struct {
	ID       string `json:"id"`        // 确认 UUID
	FilePath string `json:"file_path"` // 计划文件路径
	Content  string `json:"content"`   // 计划内容
}

// planConfirmMsg SSE plan_confirm 事件解析后的消息。
type planConfirmMsg struct {
	event *PlanConfirmEvent
}

// handlePlanConfirm 处理计划确认事件：进入全屏计划确认模式。
func (m *Model) handlePlanConfirm(msg planConfirmMsg) (tea.Model, tea.Cmd) {
	m.showingPlanConfirm = true
	m.planConfirmID = msg.event.ID
	m.planConfirmPath = msg.event.FilePath
	m.planConfirmContent = msg.event.Content
	m.planViewContent = msg.event.Content
	m.planViewport.SetContent(msg.event.Content)
	m.planViewport.GotoTop()
	return m, nil
}

// confirmPlanCmd 向服务端发送计划确认结果（confirm/deny）。
func confirmPlanCmd(client *client.Client, id, action string) tea.Cmd {
	return func() tea.Msg {
		if err := client.ToolConfirm.Confirm(id, action); err != nil {
			return systemMsg{err: err}
		}
		return nil
	}
}
