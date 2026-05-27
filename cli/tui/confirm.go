package tui

import (
	"mifer/cli/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// confirm.go — 工具调用确认类型与逻辑
// ============================================================================
//
// 当 Agent 执行未在白名单中的工具调用时，TUI 弹出确认对话框。
// 用户可通过选择列表选择接受、拒绝或始终允许。
//
// 流程：
//   1. SSE 收到 tool_confirm 事件 → toolConfirmMsg → handleToolConfirm 进入确认模式
//   2. 用户 ↑↓ 导航选择 → Enter 确认 → confirmCmd 异步发送 HTTP POST
//   3. 服务端 ConfirmBus 解除阻塞 → 工具继续执行或返回错误

// toolConfirmMsg SSE tool_confirm 事件到达，显示确认对话框
type toolConfirmMsg struct {
	callID string // 确认调用唯一标识
	name   string // 工具名
	args   string // 工具参数（JSON）
}

// ============================================================================
// confirmOption — list.Item 实现，用于确认选择列表
// ============================================================================

// confirmOption 实现 bubbles/list.Item 接口，表示一个确认选项。
type confirmOption struct {
	label  string // 显示文本
	action string // "accept" | "refuse" | "allow"
}

func (o confirmOption) Title() string       { return o.label }
func (o confirmOption) Description() string { return "" }
func (o confirmOption) FilterValue() string { return o.label }

// ============================================================================
// Update() 中的确认消息处理器
// ============================================================================

// handleToolConfirm 处理 tool_confirm 事件 → 进入确认模式
//
// 工具名和参数以 system 消息形式显示在主 viewport 中（空间充足，避免侧边栏溢出），
// 侧边栏仅显示确认选项列表。
func (m *Model) handleToolConfirm(msg toolConfirmMsg) (tea.Model, tea.Cmd) {
	// 将确认信息追加到主对话显示区
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
		confirmOption{label: "接受 (仅本次)", action: "accept"},
		confirmOption{label: "拒绝", action: "refuse"},
		confirmOption{label: "始终允许 (加入白名单)", action: "allow"},
	}
	m.confirmingTool = true
	m.confirmCallID = msg.callID
	m.confirmToolName = msg.name
	m.confirmToolArgs = msg.args
	m.confirmList.ResetSelected()
	return m, tea.Batch(m.confirmList.SetItems(items), listenStreamCmd(m.streamCh))
}

// handleConfirmSelect 处理确认选择列表中的 Enter 键
//
// 获取当前选中的选项，发送 HTTP 确认到服务端，退出确认模式。
func (m *Model) handleConfirmSelect() (tea.Model, tea.Cmd) {
	action := "refuse" // 默认拒绝（如列表为空）
	if item := m.confirmList.SelectedItem(); item != nil {
		action = item.(confirmOption).action
	}
	m.confirmingTool = false
	return m, confirmCmd(m.client, m.confirmCallID, action)
}

// ============================================================================
// 确认操作命令
// ============================================================================

// confirmCmd 异步发送工具确认决定到服务端
func confirmCmd(client *client.Client, callID, action string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Confirm.Confirm(callID, action); err != nil {
			return systemMsg{err: err}
		}
		return nil // 静默成功，不追加消息
	}
}
