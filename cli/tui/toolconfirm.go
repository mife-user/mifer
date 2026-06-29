package tui

// ============================================================================
// toolconfirm.go — 工具确认类型与逻辑
// ============================================================================
//
// 工具确认使用 bubbles/list 选择列表（与 /excmem 相同模式），在侧边栏底部渲染。
// 三个选项：Yes（执行）、No（拒绝）、Allow（始终允许）。
//
// 流程：
//   1. SSE "tool_confirm" 事件 → toolConfirmMsg → handleToolConfirm
//   2. handleToolConfirm → 加入 confirmQueue → 进入选择模式
//   3. 用户按键导航 → Enter 选择 → handleConfirmSelect 执行对应操作
//   4. 异步 HTTP 请求 → 服务端 resolve channel → 中间件解阻塞

import (
	"fmt"
	"time"

	"mifer/cli/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ToolConfirmPrompt 单个待确认工具调用，由 SSE tool_confirm 事件产生。
type ToolConfirmPrompt struct {
	ID        string `json:"id"`        // 确认 UUID
	ToolName  string `json:"tool_name"` // 工具名称
	Arguments string `json:"arguments"` // 参数 JSON
}

// ============================================================================
// confirmItem — list.Item 实现，用于确认选择列表
// ============================================================================

type confirmItem struct {
	label       string
	description string
}

func (i confirmItem) Title() string       { return i.label }
func (i confirmItem) Description() string { return i.description }
func (i confirmItem) FilterValue() string { return i.label }

// newConfirmItems 构建确认选择列表项，通过构造函数管理数据定义。
func newConfirmItems() []list.Item {
	return []list.Item{
		confirmItem{label: "Yes", description: "确认执行本次工具调用"},
		confirmItem{label: "No", description: "拒绝本次工具调用"},
		confirmItem{label: "Allow", description: "始终允许，后续不再询问（命令加入白名单）"},
	}
}

// ============================================================================
// Bubble Tea 消息类型
// ============================================================================

// toolConfirmMsg SSE tool_confirm 事件解析后进入 Update 的消息。
// 需要用户确认的工具调用。
type toolConfirmMsg struct {
	prompt *ToolConfirmPrompt
}

// toolAutoConfirmMsg sessionAllowed 命中时自动确认的消息。
type toolAutoConfirmMsg struct {
	prompt *ToolConfirmPrompt
}

// ============================================================================
// Update() 中的确认消息处理器
// ============================================================================

// handleToolConfirm 处理需要确认的工具调用：
// 1. 在对话框中添加 system 消息显示工具名+参数
// 2. 加入 confirmQueue
// 3. 若未处于选择模式，进入选择模式展示列表
func (m *Model) handleToolConfirm(msg toolConfirmMsg) (tea.Model, tea.Cmd) {
	prompt := msg.prompt

	// 在对话框中显示确认信息
	displayMsg := fmt.Sprintf("🔧 确认执行工具: %s", prompt.ToolName)
	if prompt.Arguments != "" && prompt.Arguments != "{}" {
		formatted := formatConfirmArgs(prompt.ToolName, prompt.Arguments)
		if formatted != "" {
			displayMsg += "\n" + formatted
		}
	}
	m.messages = append(m.messages, message{
		role:    "system",
		content: displayMsg,
	})
	m.needsAutoScroll = true

	// 加入队列
	m.confirmQueue = append(m.confirmQueue, prompt)

	// 如果还未进入选择模式，展示第一个
	if !m.selectingConfirm && len(m.confirmQueue) == 1 {
		return m.startConfirmSelection()
	}

	if m.streamCh != nil {
		return m, listenStreamCmd(m.streamCh)
	}
	return m, nil
}

// handleToolAutoConfirm 处理 sessionAllowed 中已允许的工具：自动确认，跳过用户交互。
func (m *Model) handleToolAutoConfirm(msg toolAutoConfirmMsg) (tea.Model, tea.Cmd) {
	return m, confirmToolCmd(m.client, msg.prompt.ID, "confirm")
}

// startConfirmSelection 进入确认选择模式：复用预初始化的 confirmList，仅替换数据项。
func (m *Model) startConfirmSelection() (tea.Model, tea.Cmd) {
	if len(m.confirmQueue) == 0 {
		return m, nil
	}

	m.selectingConfirm = true
	m.currentConfirm = m.confirmQueue[0]

	// 复用预初始化的 confirmList，仅替换数据项
	return m, m.confirmList.SetItems(newConfirmItems())
}

// handleConfirmSelect 处理选择列表中的 Enter 键。
func (m *Model) handleConfirmSelect() (tea.Model, tea.Cmd) {
	item := m.confirmList.SelectedItem()
	if item == nil {
		return m.cancelConfirm()
	}

	ci := item.(confirmItem)
	prompt := m.currentConfirm
	if prompt == nil {
		return m.cancelConfirm()
	}

	var cmd tea.Cmd
	switch ci.label {
	case "Yes":
		cmd = confirmToolCmd(m.client, prompt.ID, "confirm")
	case "No":
		cmd = confirmToolCmd(m.client, prompt.ID, "deny")
		m.messages = append(m.messages, message{
			role:    "system",
			content: "✗ 已拒绝工具调用: " + prompt.ToolName,
		})
		m.needsAutoScroll = true
	case "Allow":
		// 非命令工具加入 session 白名单
		if prompt.ToolName != "command_executor" {
			m.sessionAllowed[prompt.ToolName] = true
		} else {
			// 命令工具：解析命令并加入白名单文件
			cmd_ := parseCommandForAllowlist(prompt.Arguments)
			if cmd_ != "" {
				// 异步添加到 allowlist，带超时防止 goroutine 泄漏
				go func() {
					done := make(chan struct{}, 1)
					go func() {
						_ = m.client.AllowlistAdd.Add(cmd_)
						done <- struct{}{}
					}()
					select {
					case <-done:
					case <-time.After(10 * time.Second):
					}
				}()
			}
		}
		cmd = confirmToolCmd(m.client, prompt.ID, "allow")
		m.messages = append(m.messages, message{
			role:    "system",
			content: "✓ 已允许工具: " + prompt.ToolName + "（后续自动确认）",
		})
		m.needsAutoScroll = true
	}

	// 出队，展示下一个
	m.confirmQueue = m.confirmQueue[1:]
	m.currentConfirm = nil

	if len(m.confirmQueue) > 0 {
		// 还有待确认项，继续展示，同时发出当前确认的 HTTP 请求
		nextModel, nextCmd := m.startConfirmSelection()
		return nextModel, tea.Batch(cmd, nextCmd)
	}

	// 队列为空，退出选择模式
	m.selectingConfirm = false
	if m.streamCh != nil {
		return m, tea.Batch(cmd, listenStreamCmd(m.streamCh))
	}
	return m, cmd
}

// cancelConfirm 取消确认（Esc 等同于 No）。
func (m *Model) cancelConfirm() (tea.Model, tea.Cmd) {
	if m.currentConfirm == nil {
		m.selectingConfirm = false
		m.confirmQueue = nil
		return m, nil
	}

	prompt := m.currentConfirm
	m.confirmQueue = m.confirmQueue[1:]
	m.currentConfirm = nil

	cmd := confirmToolCmd(m.client, prompt.ID, "deny")
	m.messages = append(m.messages, message{
		role:    "system",
		content: "✗ 已取消工具调用: " + prompt.ToolName,
	})
	m.needsAutoScroll = true

	if len(m.confirmQueue) > 0 {
		nextModel, nextCmd := m.startConfirmSelection()
		return nextModel, tea.Batch(cmd, nextCmd)
	}

	m.selectingConfirm = false
	if m.streamCh != nil {
		return m, tea.Batch(cmd, listenStreamCmd(m.streamCh))
	}
	return m, cmd
}

// confirmToolCmd 发送工具确认 HTTP 请求（异步）。
func confirmToolCmd(client *client.Client, id, action string) tea.Cmd {
	return func() tea.Msg {
		if err := client.ToolConfirm.Confirm(id, action); err != nil {
			return systemMsg{err: err}
		}
		return nil
	}
}
