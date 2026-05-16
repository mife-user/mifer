package tui

import (
	"context"
	"strings"
	"time"

	"mifer/cli/client"

	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// Update — Bubble Tea 核心消息分发器
// ============================================================================
// 消息处理优先级：
//   1. WindowSizeMsg  → 布局重算（缓存 contentHeight）
//   2. MouseMsg       → 滚轮滚动
//   3. KeyMsg         → 按键处理（见 key 处理分支）
//   4. chatRespMsg    → AI 响应（markdown 渲染 + 追加消息）
//   5. systemMsg      → 系统命令结果
//   6. thinkingTickMsg → 旋转动画帧推进

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

		// ---- 窗口尺寸变化：重算布局 ----
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			if m.height < m.config.Cli.Tui.MinHeight && m.height > 0 {
				m.height = m.config.Cli.Tui.MinHeight
			}
			// textarea 宽度：终端宽度减去左右边距
			if m.width > m.config.Cli.Tui.ContentMargin*2 {
				m.textarea.SetWidth(m.width - m.config.Cli.Tui.ContentMargin*2)
			}
			m.adjustInputHeight()
			// 将 resize 转发给 textarea
			_, _ = m.textarea.Update(msg)
			return m, nil

		// ---- 鼠标滚轮：上下滚动视口 ----
		case tea.MouseMsg:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if m.scrollOff > 0 {
					m.scrollOff--
				}
			case tea.MouseButtonWheelDown:
				m.scrollOff++
			}
			return m, nil

		// ---- 键盘输入 ----
		case tea.KeyMsg:
			switch msg.String() {

			// 退出
			case "ctrl+c", "esc":
				return m, tea.Quit

			// Ctrl+N：在 textarea 中插入换行符（Windows 终端不支持 Ctrl/Shift/Alt+Enter）
			case "ctrl+n":
				m.textarea.InsertString("\n")
				m.adjustInputHeight()
				return m, nil

			// Enter：提交输入
			case "enter":
				return m.handleEnter()

			// ↑ 键：历史导航（上一条）
			case "up":
				if m.textarea.Line() == 0 {
					return m.handleHistoryUp()
				}
				m.textarea, _ = m.textarea.Update(msg)
				m.adjustInputHeight()
				return m, nil

			// ↓ 键：仅在末行时触发历史导航（下一条）
			case "down":
				if m.textarea.Line() == m.textarea.LineCount()-1 {
					return m.handleHistoryDown()
				}
				m.textarea, _ = m.textarea.Update(msg)
				m.adjustInputHeight()
				return m, nil

			// Tab：命令补全
			case "tab":
				return m.handleTabComplete()

			// 其他按键：转发给 textarea 处理（同时重置补全状态）
			default:
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				m.adjustInputHeight()
				// 用户手动修改输入后，重置补全状态
				if m.completionIdx != -1 && m.textarea.Value() != m.completionBase && !strings.HasPrefix(m.textarea.Value(), m.completionBase) {
					m.resetCompletion()
				}
				return m, cmd
			}

		// ---- AI 响应到达（SSE 流累积完成） ----
		case chatRespMsg:
			m.thinking = false
			if msg.err != nil {
				m.err = "错误: " + msg.err.Error()
				return m, nil
			}
			content := strings.TrimSpace(msg.content)
			if content == "" {
				m.err = "AI 返回了空内容"
				return m, nil
			}
			// 用 glamour 渲染 AI 响应的 markdown
			rendered, err := m.mark.Render(content)
			if err != nil {
				m.messages = append(m.messages, message{
					role:    "assistant",
					content: content,
				})
				m.err = "Markdown 渲染失败，显示原始内容"
				return m, nil
			}
			m.messages = append(m.messages, message{
				role:     "assistant",
				content:  content,
				rendered: rendered,
			})
			m.autoScrollToBottom()
			return m, nil

		// ---- 系统命令结果 ----
		case systemMsg:
			if msg.err != nil {
				m.err = "错误: " + msg.err.Error()
				return m, nil
			}
			m.messages = append(m.messages, message{
				role:    "system",
				content: msg.content,
			})
			m.autoScrollToBottom()
			return m, nil

		// ---- 思考动画 tick ----
		case thinkingTickMsg:
			if m.thinking {
				m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
				return m, m.thinkingTickCmd()
			}
			return m, nil
		}

	return m, nil
}

// adjustInputHeight 根据内容行数动态调整输入框高度，同步重算消息区可用行数。
// 受 textarea.MaxHeight 约束，上限 5 行。
func (m *Model) adjustInputHeight() {
	lines := max(m.textarea.LineCount(), 1)
	m.textarea.SetHeight(lines)
	m.contentHeight = max(m.height-m.textarea.Height()-1, 1)
}

// ============================================================================
// 按键处理
// ============================================================================

// handleEnter 处理 Enter 提交：读取 textarea 值 → 识别命令 → 发送聊天
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}
	m.textarea.Reset()
	m.textarea.SetHeight(1)               // 提交后重置为 1 行
	m.contentHeight = max(m.height-1-1, 1) // 恢复消息区高度
	m.err = ""

	// 记录到历史（去重：与最近一条相同则不重复追加）
	if len(m.history) == 0 || m.history[len(m.history)-1] != input {
		if len(m.history) >= m.config.Cli.Tui.MaxHistory {
			m.history = m.history[1:] // 环形淘汰最早记录
		}
		m.history = append(m.history, input)
	}
	m.historyIdx = -1
	m.pendingInput = ""
	m.resetCompletion()

	switch {
	case input == "exit" || input == "quit":
		return m, tea.Quit

	case input == "help":
		m.messages = append(m.messages, message{
			role:    "system",
			content: "命令: ↑↓ 历史输入 | Ctrl+N 换行 | /viewmemory 查看记忆 | /excmem <id> 切换会话 | exit 退出 | help 帮助",
		})
		m.autoScrollToBottom()
		return m, nil

	case strings.HasPrefix(input, "/viewmemory"):
		id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory"))
		return m, loadMemoryCmd(m.client, id)

	case strings.HasPrefix(input, "/excmem"):
		id := strings.TrimSpace(strings.TrimPrefix(input, "/excmem"))
		return m, excmemCmd(m.client, id)

	default:
		// 用户聊天消息
		m.messages = append(m.messages, message{
			role:    "user",
			content: input,
		})
		m.thinking = true
		m.autoScrollToBottom()
		return m, tea.Batch(
			sendChatCmd(m.client, input),
			m.thinkingTickCmd(),
		)
	}
}

// handleHistoryUp 处理 ↑ 键：浏览上一条历史输入
func (m *Model) handleHistoryUp() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	// 首次进入历史导航，暂存当前 textarea 内容
	if m.historyIdx == -1 {
		m.pendingInput = m.textarea.Value()
		m.historyIdx = len(m.history) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}
	// 更新 textarea 显示
	m.textarea.Reset()
	m.textarea.InsertString(m.history[m.historyIdx])
	// 光标移到末尾
	m.textarea.CursorEnd()
	m.adjustInputHeight()
	return m, nil
}

// handleHistoryDown 处理 ↓ 键：浏览下一条历史输入
func (m *Model) handleHistoryDown() (tea.Model, tea.Cmd) {
	if m.historyIdx == -1 {
		return m, nil
	}
	if m.historyIdx < len(m.history)-1 {
		m.historyIdx++
		m.textarea.Reset()
		m.textarea.InsertString(m.history[m.historyIdx])
		m.textarea.CursorEnd()
	} else {
		// 已到最新，恢复暂存内容
		m.historyIdx = -1
		m.textarea.Reset()
		m.textarea.InsertString(m.pendingInput)
		m.textarea.CursorEnd()
		m.pendingInput = ""
	}
	m.adjustInputHeight()
	return m, nil
}

// ============================================================================
// Tab 补全
// ============================================================================

// handleTabComplete 处理 Tab 键：补全内置命令。
// 补全规则：
//   - 0 个匹配 → 无操作
//   - 1 个匹配 → 直接补全到该命令
//   - N 个匹配 → 首次补全最长公共前缀，再次 Tab 循环切换
func (m *Model) handleTabComplete() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()
	trimmed := strings.TrimSpace(input)

	// 已激活补全循环且用户未修改前缀 → 切换到下一个匹配
	if m.completionIdx >= 0 && trimmed == m.completionBase {
		return m.cycleCompletion()
	}

	// 否则：计算新的匹配列表
	matches := m.findMatches(trimmed)
	switch len(matches) {
	case 0:
		return m, nil
	case 1:
		// 唯一匹配 → 直接替换
		m.textarea.Reset()
		m.textarea.InsertString(matches[0] + " ")
		m.textarea.CursorEnd()
		m.resetCompletion()
		m.adjustInputHeight()
		return m, nil
	default:
		// 多个匹配 → 补全到最长公共前缀
		common := longestCommonPrefix(matches)
		if common != trimmed {
			m.textarea.Reset()
			m.textarea.InsertString(common)
			m.textarea.CursorEnd()
		}
		// 记录补全状态，等待下次 Tab 循环
		m.completions = matches
		m.completionIdx = -1
		m.completionBase = common
		m.adjustInputHeight()
		return m, nil
	}
}

// cycleCompletion 在多个匹配中循环切换（再次 Tab 触发）
func (m *Model) cycleCompletion() (tea.Model, tea.Cmd) {
	if len(m.completions) == 0 {
		return m, nil
	}
	m.completionIdx = (m.completionIdx + 1) % len(m.completions)
	m.textarea.Reset()
	m.textarea.InsertString(m.completions[m.completionIdx] + " ")
	m.textarea.CursorEnd()
	m.adjustInputHeight()
	return m, nil
}

// findMatches 返回以 prefix 开头的可补全命令列表
func (m *Model) findMatches(prefix string) []string {
	if prefix == "" {
		return nil
	}
	lower := strings.ToLower(prefix)
	var matches []string
	for _, cmd := range m.config.Cli.Tui.CompletableCommands {
		if strings.HasPrefix(strings.ToLower(cmd), lower) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// resetCompletion 清除补全状态
func (m *Model) resetCompletion() {
	m.completions = nil
	m.completionIdx = -1
	m.completionBase = ""
}

// longestCommonPrefix 返回字符串切片的最长公共前缀
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// ============================================================================
// 滚动控制
// ============================================================================

// autoScrollToBottom 新消息到达时自动滚到底部
// 计算消息总行数，如果新增则更新 scrollOff 指向底部
func (m *Model) autoScrollToBottom() {
	// 预估消息总行数（粗略计算，View 中会精确控制）
	totalLines := m.countMessageLines()
	if totalLines != m.lastMsgLine {
		m.lastMsgLine = totalLines
		// 将 scrollOff 设到最大值，View 会自动 clamp
		m.scrollOff = totalLines
	}
}

// countMessageLines 统计消息渲染总行数
// 每条消息按 "\n" 拆分后计数，外加分隔线
func (m *Model) countMessageLines() int {
	count := 0
	for _, msg := range m.messages {
		content := msg.rendered
		if content == "" {
			content = msg.content
		}
		count += len(strings.Split(content, "\n")) // 消息内容行数
		count++                                      // 分隔线
	}
	if m.thinking {
		count++ // 旋转动画行
	}
	if m.err != "" {
		count++
	}
	return count
}

// ============================================================================
// 异步命令
// ============================================================================

// sendChatCmd 发送聊天请求，积累所有 SSE chunk 后返回完整响应
// 非流式实现：所有 chunk 在内存中累积，完成后一次性发出 chatRespMsg
func sendChatCmd(client *client.Client, content string) tea.Cmd {
	return func() tea.Msg {
		var buf strings.Builder
		ctx := context.Background()
		err := client.Chat.Send(ctx, content, func(event, chunk string) error {
			if event == "thinking" {
				return nil // 跳过 thinking 事件
			}
			buf.WriteString(chunk)
			return nil
		})
		return chatRespMsg{content: buf.String(), err: err}
	}
}

// thinkingTickCmd 思考动画 ticker：按配置的间隔发出 thinkingTickMsg
// 在 tea.Batch 中启动，通过自旋保持动画运行直到 m.thinking = false
func (m *Model) thinkingTickCmd() tea.Cmd {
	interval := time.Duration(m.config.Cli.Tui.ThinkingTickMs) * time.Millisecond
	tickCmd := tea.Tick(interval, func(_ time.Time) tea.Msg {
		return thinkingTickMsg{}
	})
	return func() tea.Msg {
		return tickCmd()
	}
}

// loadMemoryCmd 异步加载指定会话的对话记忆
func loadMemoryCmd(client *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if id == "" {
			id = "default"
		}
		memory, err := client.Memory.Load(id)
		if err != nil {
			return systemMsg{err: err}
		}
		if memory == "" {
			memory = "(暂无对话记忆)"
		}
		return systemMsg{content: "═══════ 对话记忆 ═══════\n" + memory + "\n══════════════════════════"}
	}
}

// excmemCmd 异步切换记忆会话
func excmemCmd(client *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Excmem.Exchange(id); err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "已切换到记忆会话: " + id}
	}
}
