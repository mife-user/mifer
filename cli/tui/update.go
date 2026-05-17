package tui

// ============================================================================
// update.go — Bubble Tea 消息分发器与业务逻辑
// ============================================================================
//
// Update() 是 Bubble Tea 的核心——所有事件都经过此方法分发处理。
// 签名：(tea.Model, tea.Cmd) 表示返回新状态和一个可选的后继命令。
//
// 消息处理优先级（按 switch 顺序）：
//   1. tea.WindowSizeMsg  → 终端尺寸变化，重新计算布局
//   2. tea.MouseMsg       → 鼠标事件，委托给 viewport 处理滚轮
//   3. tea.KeyMsg         → 键盘输入，Enter/↑/↓/Tab/Ctrl+C 等
//   4. chatRespMsg        → AI 对话响应，markdown 渲染并追加到消息列表
//   5. systemMsg          → 系统命令结果（/viewmemory、/excmem）
//   6. spinner.TickMsg    → 旋转动画帧推进（bubbles/spinner 内部定时器）
//
// 按键处理策略：
//   - ↑ 键仅在 textarea 第一行时触发历史导航，否则转发给 textarea
//   - ↓ 键仅在 textarea 最后一行时触发历史导航，否则转发给 textarea
//   - 这样用户可以正常在 textarea 内使用 ↑↓ 移动光标（非首/末行时）
//
// 异步命令模型（tea.Cmd）：
//   tea.Cmd 是 func() tea.Msg 的别名。所有耗时操作（HTTP 请求、定时器）
//   都封装为 Cmd 返回，由 Bubble Tea 框架在后台执行，完成后将结果通过
//   tea.Msg 送回 Update()。这保证了 UI 始终响应，不会被阻塞。

import (
	"context"
	"strings"

	"mifer/cli/client"

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
	// 终端尺寸变化时（包括首次启动），Bubble Tea 发送 WindowSizeMsg。
	// 需要同步更新 textarea 和 viewport 的宽高。
	// 注意：MinHeight 约束防止窗口过小时 UI 崩溃。
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.height < m.config.Cli.Tui.MinHeight && m.height > 0 {
			m.height = m.config.Cli.Tui.MinHeight
		}
		// textarea 宽度：终端宽度减去内容边距（左右各 ContentMargin）
		if m.width > m.config.Cli.Tui.ContentMargin*2 {
			m.textarea.SetWidth(m.width - m.config.Cli.Tui.ContentMargin*2)
		}
		// 根据 textarea 行数动态计算消息区高度
		m.adjustInputHeight()
		// viewport 宽度减去侧边栏空间：侧边栏 1/4 + 间隙 1 + 边框内边距 2
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
		// 将 resize 事件转发给子组件，让它们自行调整内部布局
		_, _ = m.textarea.Update(msg)
		_, _ = m.viewport.Update(msg)
		return m, nil

	// ======================================================================
	// 2. 鼠标事件 → 委托给 viewport
	// ======================================================================
	// viewport.Update() 内部处理滚轮事件（MouseWheelUp/Down），
	// 每次滚动 MouseWheelDelta 行（此处配置为 1 行）。
	// 不需要额外判断——viewport 自己知道是否已到顶部/底部边界。
	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	// ======================================================================
	// 3. 键盘输入 → 按键分发
	// ======================================================================
	case tea.KeyMsg:
		switch msg.String() {

		// ---- 退出：Ctrl+C 或 Esc ----
		case "ctrl+c", "esc":
			return m, tea.Quit // tea.Quit 是特殊命令，告诉框架停止事件循环

		// ---- Ctrl+N：插入换行符（Windows 终端不支持 Ctrl/Alt+Enter） ----
		case "ctrl+n":
			m.textarea.InsertString("\n")
			m.adjustInputHeight()
			return m, nil

		// ---- Enter：提交输入（进入核心业务逻辑 handleEnter） ----
		case "enter":
			return m.handleEnter()

		// ---- ↑ 键：首行时触发历史导航，否则光标上移 ----
		// 这是为了实现"在 textarea 内可以正常移动光标，到边界时进入历史"
		case "up":
			if m.textarea.Line() == 0 {
				return m.handleHistoryUp()
			}
			m.textarea, _ = m.textarea.Update(msg)
			m.adjustInputHeight()
			return m, nil

		// ---- ↓ 键：末行时触发历史导航，否则光标下移 ----
		case "down":
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
		// textarea 处理所有可见字符、退格、删除、Home/End 等
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			m.adjustInputHeight()
			// 用户手动修改输入后，重置补全状态（补全上下文不再有效）
			if m.completionIdx != -1 && m.textarea.Value() != m.completionBase && !strings.HasPrefix(m.textarea.Value(), m.completionBase) {
				m.resetCompletion()
			}
			return m, cmd
		}

	// ======================================================================
	// 4a. 流式状态更新（agent切换、工具调用）
	// ======================================================================
	// startSSECmd 在后台 goroutine 中通过 SSE 回调分发状态变化到 streamCh。
	// 每次收到状态更新后递归监听下一条消息，保持流式通道活跃。
	case streamStatusMsg:
		m.sidebar.update(msg)
		if m.streamCh != nil {
			return m, listenStreamCmd(m.streamCh)
		}
		return m, nil

	// ======================================================================
	// 4b. 流式内容片段
	// ======================================================================
	// 每收到一段 response 文本就追加到累积缓冲区。
	case streamContentMsg:
		if m.accBuf != nil {
			m.accBuf.WriteString(msg.content)
		}
		if m.streamCh != nil {
			return m, listenStreamCmd(m.streamCh)
		}
		return m, nil

	// ======================================================================
	// 4c. 流式传输完成 → 处理最终响应
	// ======================================================================
	// SSE 流结束（收到 [DONE] 或错误），进行 markdown 渲染并追加到消息列表。
	case streamDoneMsg:
		m.thinking = false
		m.streamCh = nil

		if msg.err != nil {
			m.err = "错误: " + msg.err.Error()
			m.accBuf = nil
			m.sidebar = SidebarState{}
			return m, nil
		}
		content := strings.TrimSpace(m.accBuf.String())
		m.accBuf = nil
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
	// 4. AI 响应到达（SSE 流累积完成）
	// ======================================================================
	// 保留用于回退场景。
	case chatRespMsg:
		m.thinking = false
		// 错误处理：将错误信息显示为红色错误文本，不追加消息
		if msg.err != nil {
			m.err = "错误: " + msg.err.Error()
			return m, nil
		}
		content := strings.TrimSpace(msg.content)
		if content == "" {
			m.err = "AI 返回了空内容"
			return m, nil
		}
		// glamour 将 markdown 渲染为带 ANSI 颜色码的终端字符串
		rendered, err := m.mark.Render(content)
		if err != nil {
			// 渲染失败时仍追加消息，但显示原始内容
			m.messages = append(m.messages, message{
				role:    "assistant",
				content: content,
			})
			m.err = "Markdown 渲染失败，显示原始内容"
			return m, nil
		}
		// 正常追加 AI 响应（含预渲染的 ANSI 输出）
		m.messages = append(m.messages, message{
			role:     "assistant",
			content:  content,
			rendered: rendered,
		})
		// 标记需要自动滚底，下一次 View() 中调用 viewport.GotoBottom()
		m.needsAutoScroll = true
		return m, nil

	// ======================================================================
	// 5. 系统命令结果（/viewmemory、/excmem）
	// ======================================================================
	// loadMemoryCmd / excmemCmd 异步完成后发出 systemMsg。
	// 与 chatRespMsg 的区别：内容不经过 markdown 渲染，直接以原始文本显示。
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
	// 6. 旋转动画帧推进（spinner 内部 tick）
	// ======================================================================
	// bubbles/spinner 每隔 ~83ms 发出一次 TickMsg。
	// 仅当 m.thinking == true 时才推进动画帧（返回下一个 tick 命令）。
	// 当 m.thinking == false 时返回 nil，spinner 动画自然停止。
	//
	// 为什么不用额外的"停止"机制？
	//   spinner 的 Update 方法总是返回下一个 tick 命令。
	//   如果我们在 thinking=false 时不返回这个命令，动画就停止了。
	//   这比之前手动维护的 thinkingTickCmd 更简洁。
	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd // cmd 是下一个 spinner.Tick() 命令，保持动画继续
		}
		return m, nil // thinking 为 false，返回 nil 停止动画
	}

	return m, nil
}

// ============================================================================
// adjustInputHeight — 动态调整输入区高度
// ============================================================================
// 每次 textarea 行数变化时调用（输入内容增多、插入换行等）。
//
// 计算逻辑：
//
//	contentHeight = 终端高度 - textarea 实际高度 - 1（间距）
//	textarea 高度由行数决定，受 MaxHeight=5 约束
//
// 例如终端 40 行，textarea 3 行：
//
//	contentHeight = 40 - 3 - 1 = 36 行用于消息显示
func (m *Model) adjustInputHeight() {
	lines := max(m.textarea.LineCount(), 1)
	m.textarea.SetHeight(lines)
	m.contentHeight = max(m.height-m.textarea.Height()-1, 1)
}

// ============================================================================
// handleEnter — Enter 键处理：核心用户交互入口
// ============================================================================
// 按输入内容分发到不同处理逻辑：
//
//	空输入      → 无操作
//	exit/quit   → 返回 tea.Quit 退出程序
//	help        → 追加系统帮助消息
//	/viewmemory → 异步加载对话记忆并显示
//	/excmem     → 异步切换记忆会话
//	其他        → 作为聊天消息发送给 AI
//
// 每次提交后：
//  1. 清空 textarea
//  2. 重置 textarea 高度为 1 行
//  3. 记录到输入历史（去重）
//  4. 重置补全状态
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	// ---- 重置输入区域 ----
	m.textarea.Reset()
	m.textarea.SetHeight(1)
	m.contentHeight = max(m.height-1-1, 1)
	m.viewport.Height = m.contentHeight // 同步 viewport 高度
	m.err = ""

	// ---- 记录到输入历史（去重） ----
	// 与最近一条相同则不重复追加，避免连续相同的输入占满历史
	if len(m.history) == 0 || m.history[len(m.history)-1] != input {
		if len(m.history) >= m.config.Cli.Tui.MaxHistory {
			m.history = m.history[1:] // 环形淘汰最早记录
		}
		m.history = append(m.history, input)
	}
	m.historyIdx = -1   // 退出历史导航
	m.pendingInput = "" // 清空暂存
	m.resetCompletion() // 重置补全状态

	// ---- 命令分发 ----
	switch {
	case input == "exit" || input == "quit":
		return m, tea.Quit

	case input == "help":
		// 帮助信息作为 system 消息追加，不发送到后端
		m.messages = append(m.messages, message{
			role:    "system",
			content: "命令: ↑↓ 历史输入 | Ctrl+N 换行 | /viewmemory 查看记忆 | /excmem <id> 切换会话 | exit 退出 | help 帮助",
		})
		m.needsAutoScroll = true
		return m, nil

	case strings.HasPrefix(input, "/viewmemory"):
		// 异步加载记忆：loadMemoryCmd 通过 HTTP GET 获取指定会话的对话历史
		// 完成后发出 systemMsg，在 Update() 中追加为 system 消息
		id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory"))
		return m, loadMemoryCmd(m.client, id)

	case strings.HasPrefix(input, "/excmem"):
		// 异步切换会话：excmemCmd 通过 HTTP POST 切换到指定记忆会话
		id := strings.TrimSpace(strings.TrimPrefix(input, "/excmem"))
		return m, excmemCmd(m.client, id)

	default:
		// ---- 用户聊天消息 ----
		// 1. 追加 user message 到消息列表
		// 2. 设置 thinking=true → View() 开始渲染 spinner
		// 3. 初始化流式传输状态（channel + 累积缓冲区 + 侧边栏重置）
		// 4. 同时启动三个异步命令：
		//    - startSSECmd：HTTP SSE 请求，逐事件推送到 channel
		//    - listenStreamCmd：从 channel 读取事件送入 Update
		//    - spinner.Tick：启动旋转动画（每 ~83ms 一帧）
		m.messages = append(m.messages, message{
			role:    "user",
			content: input,
		})
		m.thinking = true
		m.needsAutoScroll = true
		// 初始化流式传输
		m.accBuf = &strings.Builder{}
		m.streamCh = make(chan tea.Msg, 32)
		m.sidebar = SidebarState{}
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(m.spinner.Tick())
		return m, tea.Batch(
			startSSECmd(m.client, input, m.streamCh),
			listenStreamCmd(m.streamCh),
			spCmd,
		)
	}
}

// ============================================================================
// handleHistoryUp / handleHistoryDown — 历史输入导航
// ============================================================================
//
// 交互设计：
//   首次按 ↑（在首行）→ 暂存当前 textarea 内容到 pendingInput
//                    → 显示最近一条历史记录
//   继续按 ↑        → 显示更早的历史记录
//   按 ↓             → 显示更新的历史记录
//   到达最新后按 ↓   → 恢复 pendingInput（退出历史导航前的内容）
//
// 为什么需要 pendingInput？
//   用户在输入框中写了一半的内容，按 ↑ 查看历史，再按 ↓ 回到最新时，
//   应该恢复之前未发送的内容，而不是显示空输入框。

func (m *Model) handleHistoryUp() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	// 首次进入历史导航：暂存当前 textarea 内容
	if m.historyIdx == -1 {
		m.pendingInput = m.textarea.Value()
		m.historyIdx = len(m.history) - 1 // 从最新一条开始
	} else if m.historyIdx > 0 {
		m.historyIdx-- // 向更早的历史移动
	}
	m.textarea.Reset()
	m.textarea.InsertString(m.history[m.historyIdx])
	m.textarea.CursorEnd()
	m.adjustInputHeight()
	return m, nil
}

func (m *Model) handleHistoryDown() (tea.Model, tea.Cmd) {
	if m.historyIdx == -1 {
		return m, nil // 不在历史导航中，无操作
	}
	if m.historyIdx < len(m.history)-1 {
		m.historyIdx++ // 向更新的历史移动
		m.textarea.Reset()
		m.textarea.InsertString(m.history[m.historyIdx])
		m.textarea.CursorEnd()
	} else {
		// 已到达最新一条历史 → 恢复暂存的用户输入
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
//
// 补全策略（三级）：
//
//	0 个匹配    → 无操作
//	1 个匹配    → 直接替换 textarea 内容为该命令 + 空格
//	N 个匹配    → 首次 Tab：补全到最长公共前缀
//	            → 再次 Tab：循环切换到下一个匹配
//
// 示例（假设可补全命令有 /viewmemory, /excmem, exit, help）：
//
//	输入 /vi [Tab]    → 唯一匹配 /viewmemory → 直接补全
//	输入 /v [Tab]     → 匹配 /viewmemory → 直接补全
//	输入 e [Tab]      → 匹配 exit → 补全到 "exit "
//	输入 / [Tab]      → 匹配 /viewmemory, /excmem → 无公共前缀（/ 之后不同）
func (m *Model) handleTabComplete() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()
	trimmed := strings.TrimSpace(input)

	// 补全循环已激活且用户未修改输入 → 切换到下一个匹配
	if m.completionIdx >= 0 && trimmed == m.completionBase {
		return m.cycleCompletion()
	}

	// 从头计算匹配列表
	matches := m.findMatches(trimmed)
	switch len(matches) {
	case 0:
		return m, nil // 无匹配，不操作
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
		// 例如 matches = ["/viewmemory", "/viewmodel"]
		// 最长公共前缀 = "/viewmem"，输入 "/vi" → 补全到 "/viewmem"
		common := longestCommonPrefix(matches)
		if common != trimmed {
			m.textarea.Reset()
			m.textarea.InsertString(common)
			m.textarea.CursorEnd()
		}
		// 记录补全状态，等待下次 Tab 循环切换
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

// findMatches 返回以 prefix 开头的可补全命令列表（大小写不敏感）
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
//
// 算法：取第一个字符串为初始前缀，逐个与后续字符串比较，
// 不断缩短前缀直到所有字符串都以它开头。
// 示例：["hello", "help", "helicopter"] → "hel"
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
// startSSECmd — 启动 SSE 流式请求，将事件写入通道
// ============================================================================
//
// 返回的 tea.Cmd 在后台 goroutine 中执行 SSE 请求。
// 通过传入的 channel 逐条推送流式消息回 Bubble Tea 事件循环。
//
// 流式消息分发逻辑：
//
//	agent_start/agent_end/tool_start/tool_end → streamStatusMsg（侧边栏更新）
//	response → streamContentMsg（累积到 accBuf）
//	thinking → 跳过
//
// goroutine 退出前关闭 channel，触发 listenStreamCmd 返回 nil 停止递归。
func startSSECmd(client *client.Client, content string, ch chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer close(ch)
			ctx := context.Background()
			err := client.Chat.Send(ctx, content, func(event, chunk string) error {
				switch event {
				case "agent_start", "agent_end", "tool_start", "tool_end":
					ch <- streamStatusMsg{event: event, name: chunk}
				case "thinking":
					// 跳过 thinking 事件
				case "response":
					ch <- streamContentMsg{content: chunk}
				}
				return nil
			})
			ch <- streamDoneMsg{err: err}
		}()
		return nil // nil msg 被 Bubble Tea 忽略
	}
}

// listenStreamCmd 从通道读取下一条流式消息的递归命令
//
// Bubble Tea 的标准"持续监听"模式：
//
//	收到消息 → 在 Update 中处理后返回 listenStreamCmd(ch)
//	→ 阻塞等待下一条消息 → 收到后再次进入 Update → 循环
//	通道关闭时返回 nil（递归终止），后续不再有 stream 消息
func listenStreamCmd(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil // 通道关闭，递归终止
		}
		return msg
	}
}

// loadMemoryCmd 异步加载指定会话的对话记忆
//
// 通过 HTTP GET /api/memory/:id 获取 JSONL 格式的对话历史。
// id 为空时默认加载 "default" 会话。
// 响应内容直接作为 systemMsg 显示，包含分隔线装饰。
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
//
// 通过 HTTP POST /api/memory/exchange 切换到指定的记忆会话。
// 切换后后续对话将读写该会话的记忆文件。
func excmemCmd(client *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Excmem.Exchange(id); err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "已切换到记忆会话: " + id}
	}
}
