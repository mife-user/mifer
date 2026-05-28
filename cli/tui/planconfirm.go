package tui

import (
	"mifer/cli/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// planConfirmMsg SSE plan_confirm 事件到达
type planConfirmMsg struct {
	callID  string
	content string
}

// planConfirmOption 计划确认选项
type planConfirmOption struct {
	label  string
	action string // "accept" | "refuse" | "supplement"
}

func (o planConfirmOption) Title() string       { return o.label }
func (o planConfirmOption) Description() string { return "" }
func (o planConfirmOption) FilterValue() string { return o.label }

// handlePlanConfirm 处理 plan_confirm 事件 → 进入计划确认模式
func (m *Model) handlePlanConfirm(msg planConfirmMsg) (tea.Model, tea.Cmd) {
	rendered, err := m.mark.Render(msg.content)
	content := msg.content
	if err == nil && rendered != "" {
		content = rendered
	}
	m.messages = append(m.messages, message{
		role:    "assistant",
		content: content,
	})
	m.needsAutoScroll = true

	items := []list.Item{
		planConfirmOption{label: "接受 (执行计划)", action: "accept"},
		planConfirmOption{label: "拒绝", action: "refuse"},
		planConfirmOption{label: "补充 (填写意见后重新制定)", action: "supplement"},
	}
	m.confirmingPlan = true
	m.planConfirmCallID = msg.callID
	m.planList.ResetSelected()
	return m, tea.Batch(m.planList.SetItems(items), listenStreamCmd(m.streamCh))
}

// handlePlanConfirmSelect 处理计划确认选择列表中的 Enter 键
func (m *Model) handlePlanConfirmSelect() (tea.Model, tea.Cmd) {
	action := "refuse"
	if item := m.planList.SelectedItem(); item != nil {
		action = item.(planConfirmOption).action
	}

	if action == "supplement" {
		m.planSupplement = true
		m.planConfirmAction = "supplement"
		m.pendingPlanInput = m.textarea.Value()
		m.textarea.Reset()
		m.textarea.Placeholder = "请输入补充意见，然后按 Enter 发送..."
		m.textarea.Focus()
		m.adjustInputHeight()
		return m, listenStreamCmd(m.streamCh)
	}

	m.confirmingPlan = false
	return m, tea.Batch(planConfirmCmd(m.client, m.planConfirmCallID, action, ""), listenStreamCmd(m.streamCh))
}

// handlePlanSupplementSubmit 提交补充意见
func (m *Model) handlePlanSupplementSubmit() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()
	m.planSupplement = false
	m.confirmingPlan = false
	m.textarea.Reset()
	m.textarea.Placeholder = ""
	m.textarea.InsertString(m.pendingPlanInput)
	m.adjustInputHeight()

	supplement := input
	if supplement == "" {
		supplement = "请重新制定计划"
	}
	return m, tea.Batch(planConfirmCmd(m.client, m.planConfirmCallID, "supplement", supplement), listenStreamCmd(m.streamCh))
}

// planConfirmCmd 异步发送计划确认决定到服务端
func planConfirmCmd(client *client.Client, callID, action, input string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Confirm.PlanConfirm(callID, action, input); err != nil {
			return systemMsg{err: err}
		}
		return nil
	}
}
