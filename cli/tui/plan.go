package tui

import (
	"mifer/cli/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// plan.go — /plan 命令实现
// ============================================================================
//
// 支持两种模式：
//   1. /plan <说明>  — 发送带计划指令前缀的聊天消息给 AI
//   2. /plan（无参数）— 列出 .mifer/plans/ 下的计划文件，选择后全屏查看
//
// 消息流转（无参数模式）：
//   handleEnter() → listPlansCmd → HTTP GET /api/plan
//     → planListMsg → handlePlanList → 侧边栏选择模式
//     → 用户按 Enter → handlePlanSelect → loadPlanCmd → HTTP GET /api/plan/:name
//     → planViewMsg → handlePlanView → 全屏查看模式

// ============================================================================
// 消息类型
// ============================================================================

// planListMsg 异步获取计划文件列表的结果
type planListMsg struct {
	files []string // 计划文件名列表
	err   error    // 网络或解析错误
}

// planViewMsg 计划文件加载完成，进入全屏查看模式
type planViewMsg struct {
	name    string // 文件名
	content string // 文件内容文本
	err     error  // 网络或解析错误
}

// ============================================================================
// list.Item 实现 — planItem
// ============================================================================

// planItem 实现 bubbles/list.Item 接口，表示一个可选计划文件
type planItem struct {
	name string // 文件名（含 .md 后缀）
}

func (i planItem) Title() string       { return i.name }
func (i planItem) Description() string { return "" }
func (i planItem) FilterValue() string { return i.name }

// ============================================================================
// 命令 — 返回异步 tea.Cmd
// ============================================================================

// listPlansCmd 异步获取计划文件列表
func listPlansCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		files, err := client.Plan.List()
		return planListMsg{files: files, err: err}
	}
}

// loadPlanCmd 异步加载指定计划文件内容
func loadPlanCmd(client *client.Client, name string) tea.Cmd {
	return func() tea.Msg {
		content, err := client.Plan.Load(name)
		if err != nil {
			return planViewMsg{err: err}
		}
		if content == "" {
			content = "(空计划文件)"
		}
		return planViewMsg{name: name, content: content}
	}
}

// ============================================================================
// 消息处理器
// ============================================================================

// handlePlanList 处理计划列表结果 → 进入选择模式
func (m *Model) handlePlanList(msg planListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = "错误: " + msg.err.Error()
		return m, nil
	}
	if len(msg.files) == 0 {
		m.err = "没有可用的计划文件"
		return m, nil
	}
	var items []list.Item
	for _, f := range msg.files {
		items = append(items, planItem{name: f})
	}
	m.selectingPlan = true
	return m, m.planList.SetItems(items)
}

// handlePlanView 处理计划内容 → 进入全屏查看模式
func (m *Model) handlePlanView(msg planViewMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = "错误: " + msg.err.Error()
		return m, nil
	}
	m.showingPlanView = true
	m.planViewContent = msg.content
	m.planViewport.SetContent(msg.content)
	m.planViewport.GotoTop()
	return m, nil
}

// handlePlanSelect 处理计划选择列表中的 Enter 键
func (m *Model) handlePlanSelect() (tea.Model, tea.Cmd) {
	item := m.planList.SelectedItem()
	if item == nil {
		m.selectingPlan = false
		return m, nil
	}
	pi := item.(planItem)
	m.selectingPlan = false
	return m, loadPlanCmd(m.client, pi.name)
}
