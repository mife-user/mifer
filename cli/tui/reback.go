package tui

import (
	"fmt"
	"mifer/cli/client"
	"mifer/cli/client/rebackhandler"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// reback.go — 回退对话类型与逻辑
// ============================================================================
//
// /reback 命令允许用户回退到指定的对话轮次：
//
// 流程：
//   1. handleEnter() 识别 /reback → listRebackEntriesCmd 异步获取对话轮次列表
//   2. rebackListMsg → handleRebackList 填充选择列表，进入选择模式
//   3. 选择模式下 Enter → handleRebackSelect → rebackCmd 执行回退
//   4. rebackDoneMsg → handleRebackDone 清空消息 + 显示系统消息

// rebackListMsg 异步获取回退列表的结果
type rebackListMsg struct {
	entries []rebackhandler.RebackEntry // 可回退的对话轮次列表
	err     error                // 网络或解析错误
}

// rebackDoneMsg 回退执行完成的结果
type rebackDoneMsg struct {
	message string // 回退结果描述
	content string // 被删除的用户消息完整内容（用于预填输入框）
	err     error  // 网络或解析错误
}

// ============================================================================
// rebackItem — list.Item 实现，用于回退选择列表
// ============================================================================

// rebackItem 实现 bubbles/list.Item 接口，表示一条可回退的对话轮次。
type rebackItem struct {
	index   int    // 轮次序号
	summary string // 用户消息摘要
}

func (i rebackItem) Title() string {
	return fmt.Sprintf("%d. %s", i.index, i.summary)
}
func (i rebackItem) Description() string { return "" }
func (i rebackItem) FilterValue() string {
	return fmt.Sprintf("%d %s", i.index, i.summary)
}

// ============================================================================
// Update() 中的回退消息处理器
// ============================================================================

// handleRebackList 处理回退列表结果 → 填充选择列表，进入选择模式
func (m *Model) handleRebackList(msg rebackListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = "错误: " + msg.err.Error()
		return m, nil
	}

	if len(msg.entries) == 0 {
		m.err = "当前会话没有可回退的对话"
		return m, nil
	}

	var items []list.Item
	for _, entry := range msg.entries {
		items = append(items, rebackItem{index: entry.Index, summary: entry.Summary})
	}
	m.selectingReback = true
	m.rebackList.SetWidth(m.width / 4)
	return m, m.rebackList.SetItems(items)
}

// handleRebackSelect 处理回退选择列表中的 Enter 键
func (m *Model) handleRebackSelect() (tea.Model, tea.Cmd) {
	item := m.rebackList.SelectedItem()
	if item == nil {
		m.selectingReback = false
		return m, nil
	}
	ri := item.(rebackItem)
	m.selectingReback = false
	return m, rebackCmd(m.client, ri.index)
}

// handleRebackDone 处理回退执行完成的结果
func (m *Model) handleRebackDone(msg rebackDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = "错误: " + msg.err.Error()
		return m, nil
	}
	// 清空 TUI 显示消息
	m.messages = nil
	// 预填输入框为被回退的用户消息内容
	m.textarea.Reset()
	if msg.content != "" {
		m.textarea.InsertString(msg.content)
		m.textarea.CursorEnd()
	}
	m.adjustInputHeight()
	m.viewport.Height = m.contentHeight
	return m, func() tea.Msg {
		return systemMsg{content: msg.message}
	}
}

// ============================================================================
// 回退操作命令
// ============================================================================

// listRebackEntriesCmd 异步获取当前会话的对话轮次列表
func listRebackEntriesCmd(client *client.Client) tea.Cmd {
	return func() tea.Msg {
		entries, err := client.Reback.List()
		return rebackListMsg{entries: entries, err: err}
	}
}

// rebackCmd 异步执行回退到指定轮次之前
func rebackCmd(client *client.Client, index int) tea.Cmd {
	return func() tea.Msg {
		msg, content, err := client.Reback.Reback(index)
		return rebackDoneMsg{message: msg, content: content, err: err}
	}
}
