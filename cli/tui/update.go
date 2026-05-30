package tui

// ============================================================================
// update.go — Bubble Tea 消息分发器与核心交互逻辑
// ============================================================================

import (
	"context"
	"mifer/pkg/conf"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// Update — 消息分发器
// ============================================================================

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ======================================================================
	// 1. 窗口尺寸变化 → 重新计算布局
	// ======================================================================
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.height < conf.GetConfig().Cli.Tui.MinHeight && m.height > 0 {
			m.height = conf.GetConfig().Cli.Tui.MinHeight
		}
		if m.width > conf.GetConfig().Cli.Tui.ContentMargin*2 {
			m.textarea.SetWidth(m.width - conf.GetConfig().Cli.Tui.ContentMargin*2)
		}
		m.adjustInputHeight()
		sidebarW := m.width / 4
		if sidebarW < 20 {
			sidebarW = 20
		}
		if sidebarW > 40 {
			sidebarW = 40
		}
		m.viewport.Width = m.width - sidebarW - 1 - 2
		if m.viewport.Width < 10 {
			m.viewport.Width = 10
		}
		m.viewport.Height = m.contentHeight
		// 侧边栏 viewport 尺寸（日志区最多占终端 1/4）
		m.sidebarVP.Width = sidebarW - 2
		m.sidebarVP.Height = m.height / 4
		if m.sidebarVP.Width < 10 {
			m.sidebarVP.Width = 10
		}
		m.memoryList.SetSize(sidebarW-4, 8)
		m.rebackList.SetSize(sidebarW-4, 8)
		m.memoryViewport.Width = m.width - 4
		m.memoryViewport.Height = m.height - 2
		m.planViewport.Width = m.width - 4
		m.planViewport.Height = m.height - 2
		_, _ = m.textarea.Update(msg)
		_, _ = m.viewport.Update(msg)
		_, _ = m.sidebarVP.Update(msg)
		_, _ = m.memoryList.Update(msg)
		_, _ = m.rebackList.Update(msg)
		_, _ = m.planList.Update(msg)
		return m, nil

	// ======================================================================
	// 2. 鼠标事件 → 委托给对应 viewport
	// ======================================================================
	case tea.MouseMsg:
		if m.showingMemoryView {
			var cmd tea.Cmd
			m.memoryViewport, cmd = m.memoryViewport.Update(msg)
			return m, cmd
		}
		if m.showingPlanView {
			var cmd tea.Cmd
			m.planViewport, cmd = m.planViewport.Update(msg)
			return m, cmd
		}
		// Alt+滚轮 / 水平滚轮 → 水平滚动
		if msg.Button == tea.MouseButtonWheelLeft || msg.Button == tea.MouseButtonWheelRight ||
			(msg.Alt && msg.Button == tea.MouseButtonWheelUp) ||
			(msg.Alt && msg.Button == tea.MouseButtonWheelDown) {
			switch msg.Button {
			case tea.MouseButtonWheelLeft, tea.MouseButtonWheelUp:
				m.viewport.ScrollLeft(conf.GetConfig().Cli.Tui.HorizontalScrollStep)
			case tea.MouseButtonWheelRight, tea.MouseButtonWheelDown:
				m.viewport.ScrollRight(conf.GetConfig().Cli.Tui.HorizontalScrollStep)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	// ======================================================================
	// 3. 键盘输入 → 按键分发
	// ======================================================================
	case tea.KeyMsg:
		// ---- 全屏记忆查看模式：仅响应 esc 和鼠标滚轮 ----
		if m.showingMemoryView {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.showingMemoryView = false
				m.memoryViewContent = ""
				return m, nil
			default:
				var cmd tea.Cmd
				m.memoryViewport, cmd = m.memoryViewport.Update(msg)
				return m, cmd
			}
		}

		// ---- 全屏计划查看模式：仅响应 esc 和鼠标滚轮 ----
		if m.showingPlanView {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.showingPlanView = false
				m.planViewContent = ""
				return m, nil
			default:
				var cmd tea.Cmd
				m.planViewport, cmd = m.planViewport.Update(msg)
				return m, cmd
			}
		}

		// ---- 记忆选择模式：拦截按键，委托给 memoryList 或处理选择 ----
		if m.selectingMem {
			switch msg.String() {
			case "enter":
				return m.handleMemorySelect()
			case "esc":
				m.selectingMem = false
				m.pendingMemCmd = ""
				return m, nil
			case "up", "down", "k", "j", "home", "end", "pgup", "pgdown":
				var cmd tea.Cmd
				m.memoryList, cmd = m.memoryList.Update(msg)
				return m, cmd
			default:
				return m, nil
			}
		}

		// ---- 计划选择模式：拦截按键，委托给 planList 或处理选择 ----
		if m.selectingPlan {
			switch msg.String() {
			case "enter":
				return m.handlePlanSelect()
			case "esc":
				m.selectingPlan = false
				return m, nil
			case "up", "down", "k", "j", "home", "end", "pgup", "pgdown":
				var cmd tea.Cmd
				m.planList, cmd = m.planList.Update(msg)
				return m, cmd
			default:
				return m, nil
			}
		}

		// ---- 回退选择模式：拦截按键，委托给 rebackList 或处理选择 ----
		if m.selectingReback {
			switch msg.String() {
			case "enter":
				return m.handleRebackSelect()
			case "esc":
				m.selectingReback = false
				return m, nil
			case "up", "down", "k", "j", "home", "end", "pgup", "pgdown":
				var cmd tea.Cmd
				m.rebackList, cmd = m.rebackList.Update(msg)
				return m, cmd
			default:
				return m, nil
			}
		}

		switch msg.String() {

		// ---- 退出/中断：Ctrl+C 或 Esc ----
		case "ctrl+c", "esc":
			// 补全列表显示时仅关闭列表
			if m.showingCompletions {
				m.showingCompletions = false
				m.resetCompletion()
				return m, nil
			}
			// 流式传输进行中：取消流而非退出
			if m.thinking && m.cancel != nil {
				m.cancel()
				m.cancel = nil
				m.messages = append(m.messages, message{
					role:    "system",
					content: "STOP",
				})
				m.needsAutoScroll = true
				return m, nil
			}
			return m, tea.Quit

		// ---- Ctrl+N：插入换行符 ----
		case "ctrl+n":
			m.textarea.InsertString("\n")
			m.adjustInputHeight()
			return m, nil

		// ---- Enter：提交输入（补全列表显示时填入选中命令） ----
		case "enter":
			if m.showingCompletions && m.completionIdx >= 0 && m.completionIdx < len(m.completions) {
				m.textarea.Reset()
				m.textarea.InsertString(m.completions[m.completionIdx] + " ")
				m.textarea.CursorEnd()
				m.resetCompletion()
				m.showingCompletions = false
				m.adjustInputHeight()
				return m, nil
			}
			return m.handleEnter()

		// ---- ↑ 键：补全列表导航 / 首行时触发历史导航 / 否则光标上移 ----
		case "up":
			if m.showingCompletions && m.completionIdx > 0 {
				m.completionIdx--
				return m, nil
			}
			if m.textarea.Line() == 0 {
				return m.handleHistoryUp()
			}
			m.textarea, _ = m.textarea.Update(msg)
			m.adjustInputHeight()
			return m, nil

		// ---- ↓ 键：补全列表导航 / 末行时触发历史导航 / 否则光标下移 ----
		case "down":
			if m.showingCompletions && m.completionIdx < len(m.completions)-1 {
				m.completionIdx++
				return m, nil
			}
			if m.textarea.Line() == m.textarea.LineCount()-1 {
				return m.handleHistoryDown()
			}
			m.textarea, _ = m.textarea.Update(msg)
			m.adjustInputHeight()
			return m, nil

		// ---- Tab：命令补全 ----
		case "tab":
			return m.handleTabComplete()

		// ---- 其他按键：转发给 textarea 处理 ----
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			m.adjustInputHeight()
			// 输入变更时检测 / 前缀，自动触发补全匹配
			if m.completionIdx != -1 && m.textarea.Value() != m.completionBase && !strings.HasPrefix(m.textarea.Value(), m.completionBase) {
				m.resetCompletion()
			}
			m.updateCompletionForInput()
			return m, cmd
		}

	// ======================================================================
	// 4a. 流式状态更新（agent切换、工具调用、token统计）
	// ======================================================================
	case streamStatusMsg:
		return m.handleStreamStatus(msg)

	// ======================================================================
	// 4b. 流式内容片段
	// ======================================================================
	case streamContentMsg:
		return m.handleStreamContent(msg)

	// ======================================================================
	// 4c. 流式传输完成
	// ======================================================================
	case streamDoneMsg:
		return m.handleStreamDone(msg)

	// ======================================================================
	// 5. AI 响应到达（回退场景）
	// ======================================================================
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
		m.needsAutoScroll = true
		return m, nil

	// ======================================================================
	// 6. 系统命令结果（/viewmemory、/excmem）
	// ======================================================================
	case systemMsg:
		if msg.err != nil {
			m.err = "错误: " + msg.err.Error()
			return m, nil
		}
		m.messages = append(m.messages, message{
			role:    "system",
			content: msg.content,
		})
		m.needsAutoScroll = true
		return m, nil

	// ======================================================================
	// 7. 记忆列表结果
	// ======================================================================
	case memoryListMsg:
		return m.handleMemoryList(msg)

	// ======================================================================
	// 8. 记忆查看结果
	// ======================================================================
	case memoryViewMsg:
		return m.handleMemoryView(msg)

	// ======================================================================
	// 9. 回退列表结果
	// ======================================================================
	case rebackListMsg:
		return m.handleRebackList(msg)

	// ======================================================================
	// 10. 回退执行完成
	// ======================================================================
	case rebackDoneMsg:
		return m.handleRebackDone(msg)

	// ======================================================================
	// 11. 计划列表结果
	// ======================================================================
	case planListMsg:
		return m.handlePlanList(msg)

	// ======================================================================
	// 12. 计划查看结果
	// ======================================================================
	case planViewMsg:
		return m.handlePlanView(msg)

	// ======================================================================
	// 13. 旋转动画帧推进
	// ======================================================================
	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// ============================================================================
// adjustInputHeight — 动态调整输入区高度
// ============================================================================
func (m *Model) adjustInputHeight() {
	lines := max(m.textarea.LineCount(), 1)
	m.textarea.SetHeight(lines)
	m.contentHeight = max(m.height-m.textarea.Height()-1, 1)
}

// ============================================================================
// updateCompletionForInput — 输入以 / 开头时自动匹配补全命令
// ============================================================================
func (m *Model) updateCompletionForInput() {
	input := strings.TrimSpace(m.textarea.Value())
	if !strings.HasPrefix(input, "/") {
		if m.showingCompletions {
			m.showingCompletions = false
			m.resetCompletion()
		}
		return
	}
	matches := m.findMatches(input)
	if len(matches) == 0 {
		if m.showingCompletions {
			m.showingCompletions = false
			m.resetCompletion()
		}
		return
	}
	m.completions = matches
	m.showingCompletions = true
	if m.completionIdx == -1 || m.completionIdx >= len(matches) {
		m.completionIdx = -1
	}
}

// ============================================================================
// handleEnter — Enter 键处理：核心用户交互入口
// ============================================================================
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	// ---- 重置输入区域 ----
	m.textarea.Reset()
	m.textarea.SetHeight(1)
	m.contentHeight = max(m.height-1-1, 1)
	m.viewport.Height = m.contentHeight
	m.err = ""

	// ---- 记录到输入历史（去重） ----
	if len(m.history) == 0 || m.history[len(m.history)-1] != input {
		if len(m.history) >= conf.GetConfig().Cli.Tui.MaxHistory {
			m.history = m.history[1:]
		}
		m.history = append(m.history, input)
	}
	m.historyIdx = -1
	m.pendingInput = ""
	m.resetCompletion()

	// ---- 命令分发 ----
	switch {
	case input == "/exit" || input == "/quit":
		return m, tea.Quit

	case input == "/help":
		m.messages = append(m.messages, message{
			role:    "system",
			content: "命令: ↑↓ 历史输入 | Ctrl+N 换行 | /viewmemory 查看记忆 | /excmem <id> 切换会话 | /reback 回退对话 | /clear 新建会话 | /prompt 查看/设置提示词 | /reload 重载配置 | /exit 退出 | /help 帮助",
		})
		m.needsAutoScroll = true
		return m, nil

	case input == "/clear":
		return m, clearCmd(m.client)

	case input == "/reload":
		return m, reloadCmd(m.client)

	case input == "/prompt":
		return m, promptGetCmd(m.client)

	case strings.HasPrefix(input, "/prompt "):
		text := strings.TrimSpace(strings.TrimPrefix(input, "/prompt"))
		if text == "reset" {
			return m, promptResetCmd(m.client)
		}
		return m, promptSetCmd(m.client, text)

	case strings.HasPrefix(input, "/viewmemory"):
		id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory"))
		return m, listMemoriesCmd(m.client, "/viewmemory", id)

	case strings.HasPrefix(input, "/excmem"):
		id := strings.TrimSpace(strings.TrimPrefix(input, "/excmem"))
		return m, listMemoriesCmd(m.client, "/excmem", id)

	case input == "/reback":
		return m, listRebackEntriesCmd(m.client)

	case input == "/mcp":
		// /mcp — 显示 MCP Server 状态
		m.messages = append(m.messages, message{
			role:    "user",
			content: input,
		})
		m.needsAutoScroll = true
		return m, mcpStatusCmd(m.client)

	case strings.HasPrefix(input, "/plan"):
		args := strings.TrimSpace(strings.TrimPrefix(input, "/plan"))
		if args != "" {
			// /plan <说明> — 构造带计划指令前缀的聊天消息发送给 AI
			chatMsg := "请先制定详细计划并等待我审批。计划说明如下：\n\n" + args
			m.messages = append(m.messages, message{
				role:    "user",
				content: input,
			})
			m.thinking = true
			m.needsAutoScroll = true
			m.accBuf = &strings.Builder{}
			m.streamCh = make(chan tea.Msg, 32)
			m.sidebar = SidebarState{}
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			var spCmd tea.Cmd
			m.spinner, spCmd = m.spinner.Update(m.spinner.Tick())
			return m, tea.Batch(
				startSSECmd(m.client, chatMsg, m.streamCh, ctx),
				listenStreamCmd(m.streamCh),
				spCmd,
			)
		}
		// /plan (无参数) — 列出计划文件
		return m, listPlansCmd(m.client)

	default:
		// ---- 用户聊天消息 ----
		m.messages = append(m.messages, message{
			role:    "user",
			content: input,
		})
		m.thinking = true
		m.needsAutoScroll = true
		m.accBuf = &strings.Builder{}
		m.streamCh = make(chan tea.Msg, 32)
		m.sidebar = SidebarState{}
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(m.spinner.Tick())
		return m, tea.Batch(
			startSSECmd(m.client, input, m.streamCh, ctx),
			listenStreamCmd(m.streamCh),
			spCmd,
		) // 启动流式传输并监听消息
	}
}

// ============================================================================
// handleHistoryUp / handleHistoryDown — 历史输入导航
// ============================================================================

func (m *Model) handleHistoryUp() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	if m.historyIdx == -1 {
		m.pendingInput = m.textarea.Value()
		m.historyIdx = len(m.history) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}
	m.textarea.Reset()
	m.textarea.InsertString(m.history[m.historyIdx])
	m.textarea.CursorEnd()
	m.adjustInputHeight()
	return m, nil
}

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
// handleTabComplete — Tab 命令补全
// ============================================================================

func (m *Model) handleTabComplete() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()
	trimmed := strings.TrimSpace(input)

	if m.completionIdx >= 0 && trimmed == m.completionBase {
		return m.cycleCompletion()
	}

	matches := m.findMatches(trimmed)
	switch len(matches) {
	case 0:
		return m, nil
	case 1:
		m.textarea.Reset()
		m.textarea.InsertString(matches[0] + " ")
		m.textarea.CursorEnd()
		m.resetCompletion()
		m.adjustInputHeight()
		return m, nil
	default:
		common := longestCommonPrefix(matches)
		if common != trimmed {
			m.textarea.Reset()
			m.textarea.InsertString(common)
			m.textarea.CursorEnd()
		}
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

// findMatches 返回以 prefix 开头的可补全命令列表（大小写不敏感），同时填充描述映射
func (m *Model) findMatches(prefix string) []string {
	if prefix == "" {
		return nil
	}
	lower := strings.ToLower(prefix)
	var matches []string
	for _, cmd := range conf.GetConfig().Cli.Tui.CompletableCommands {
		if strings.HasPrefix(strings.ToLower(cmd.Command), lower) {
			matches = append(matches, cmd.Command)
			if m.completionDescs == nil {
				m.completionDescs = make(map[string]string)
			}
			m.completionDescs[cmd.Command] = cmd.Description
		}
	}
	return matches
}

// resetCompletion 清除补全状态
func (m *Model) resetCompletion() {
	m.completions = nil
	m.completionDescs = nil
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
