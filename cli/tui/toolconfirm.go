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

	"mifer/cli/client"
	"mifer/pkg/exc"

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

// startConfirmSelection 进入确认选择模式：设置 confirmList 显示选项。
func (m *Model) startConfirmSelection() (tea.Model, tea.Cmd) {
	if len(m.confirmQueue) == 0 {
		return m, nil
	}

	m.selectingConfirm = true
	m.currentConfirm = m.confirmQueue[0]

	// 初始化 confirmList
	width := m.width / 4
	if width < 20 {
		width = 20
	}
	if width > 40 {
		width = 40
	}
	m.confirmList = list.New(newConfirmItems(), list.NewDefaultDelegate(), width-4, 5)
	m.confirmList.SetShowTitle(false)
	m.confirmList.SetShowStatusBar(false)
	m.confirmList.SetShowFilter(false)
	m.confirmList.SetShowPagination(false)
	m.confirmList.SetShowHelp(false)

	return m, nil
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
				// 异步添加到 allowlist（不阻塞返回值）
				go func() {
					_ = m.client.AllowlistAdd.Add(cmd_)
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

// ============================================================================
// 工具参数 DTO — 各工具参数字段定义，用于 JSON 反序列化与格式化展示
// ============================================================================

type cmdArgs struct {
	Command string `json:"command"`
}
type fileCreateArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}
type fileWriteArgs struct {
	FilePath  string `json:"file_path"`
	Content   string `json:"content"`
	Mode      string `json:"mode"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}
type fileReadArgs struct {
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	MaxLines  int    `json:"max_lines"`
}
type webSearchArgs struct {
	Query string `json:"query"`
}
type webFetchArgs struct {
	URL string `json:"url"`
}
type imageGenArgs struct {
	Prompt string `json:"prompt"`
	Output string `json:"output"`
}
type knowledgeSearchArgs struct {
	Query string `json:"query"`
}
type knowledgeStoreArgs struct {
	FilePath string `json:"file_path"`
}

// ============================================================================
// 工具函数
// ============================================================================

// formatConfirmArgs 根据工具类型格式化参数用于对话框展示。
func formatConfirmArgs(toolName, argsJSON string) string {
	switch toolName {
	case "command_executor":
		var a cmdArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Command != "" {
			return "   命令: " + a.Command
		}
	case "file_creator":
		var a fileCreateArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil {
			s := "   文件: " + a.FilePath
			if a.Content != "" {
				s += "\n   内容: " + a.Content
			}
			return s
		}
	case "file_writer":
		var a fileWriteArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil {
			s := fmt.Sprintf("   文件: %s\n   模式: %s", a.FilePath, a.Mode)
			if a.StartLine > 0 || a.EndLine > 0 {
				s += fmt.Sprintf(" (行 %d-%d)", a.StartLine, a.EndLine)
			}
			if a.Content != "" {
				s += "\n   内容: " + a.Content
			}
			return s
		}
	case "file_reader", "file_viewer":
		var a fileReadArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil {
			s := "   文件: " + a.FilePath
			if a.StartLine > 0 || a.MaxLines > 0 {
				s += fmt.Sprintf("\n   行范围: %d~%d", a.StartLine, a.StartLine+a.MaxLines)
			}
			return s
		}
	case "web_search":
		var a webSearchArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Query != "" {
			return "   搜索: " + a.Query
		}
	case "web_fetch":
		var a webFetchArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.URL != "" {
			return "   网址: " + a.URL
		}
	case "image_generator":
		var a imageGenArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Prompt != "" {
			s := "   提示词: " + a.Prompt
			if a.Output != "" {
				s += "\n   输出: " + a.Output
			}
			return s
		}
	case "knowledge_search":
		var a knowledgeSearchArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.Query != "" {
			return "   检索: " + a.Query
		}
	case "knowledge_store":
		var a knowledgeStoreArgs
		if exc.ExcJSONToFile(argsJSON, &a) == nil && a.FilePath != "" {
			return "   文件: " + a.FilePath
		}
	}
	// 通用降级
	var generic map[string]any
	if exc.ExcJSONToFile(argsJSON, &generic) == nil {
		c, _ := exc.ExcFileToJSON(generic)
		return "   参数: " + c
	}
	return ""
}

// parseCommandForAllowlist 从 command_executor 的参数中解析出命令字符串。
func parseCommandForAllowlist(argsJSON string) string {
	var a cmdArgs
	if exc.ExcJSONToFile(argsJSON, &a) == nil {
		return a.Command
	}
	return ""
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
