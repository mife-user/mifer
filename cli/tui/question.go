package tui

// ============================================================================
// question.go — AI 需求澄清问答界面
// ============================================================================
//
// 当 AI 调用 ask_user 工具时，SSE ask_user 事件触发全屏问题界面。
// 用户通过 ↑↓ 选择选项，Enter 确认；选择"补充说明"时进入文本输入模式。
// 与 showingMemoryView 类似，采用全屏覆盖模式拦截所有按键。

import (
	"strings"

	"mifer/pkg/logger"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ──────────────────────────── 列表项类型 ────────────────────────────

// questionItem 选项列表项，实现 list.DefaultItem 接口。
type questionItem struct {
	text  string
	isSupplement bool // 是否为"补充说明"选项
}

func (i questionItem) Title() string       { return i.text }
func (i questionItem) Description() string { return "" }
func (i questionItem) FilterValue() string { return i.text }

const supplementLabel = "📝 补充说明（自定义输入）"

// ──────────────────────────── 问题显示初始化 ────────────────────────────

// handleQuestion 处理 questionMsg，设置全屏问题界面。
func (m *Model) handleQuestion(msg questionMsg) {
	// 在对话框中追加 system 消息，告知用户 AI 在等待回答
	m.messages = append(m.messages, message{role: "system", content: "AI 正在等待你的选择：" + msg.Question})

	m.questionID = msg.ID
	m.questionContent = msg.Question
	m.questionOptions = msg.Options

	// 构建列表项（选项 + 末尾自动追加"补充说明"）
	var items []list.Item
	for _, opt := range msg.Options {
		items = append(items, questionItem{text: opt, isSupplement: false})
	}
	items = append(items, questionItem{text: supplementLabel, isSupplement: true})

	// 创建列表组件
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	m.questionList = list.New(items, delegate, m.width-8, len(items)+2)
	m.questionList.SetShowTitle(false)
	m.questionList.SetShowStatusBar(false)
	m.questionList.SetShowHelp(false)
	m.questionList.SetFilteringEnabled(false)

	m.showingQuestionView = true
	m.selectingSupplement = false
	m.needsAutoScroll = true
}

// ──────────────────────────── 按键处理 ────────────────────────────

// handleQuestionKey 处理全屏问题视图下的按键。
func (m *Model) handleQuestionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.selectingSupplement {
			// 从补充输入模式退回选项列表
			m.selectingSupplement = false
			m.supplementInput = ""
			return m, nil
		}
		// 取消问题 → 发送空回答
		return m, m.sendAnswerCmd("", false)

	case "enter":
		if m.selectingSupplement {
			// 提交补充内容
			if m.supplementInput == "" {
				return m, nil // 空输入不提交
			}
			return m, m.sendAnswerCmd(m.supplementInput, true)
		}
		// 检查是否选中了"补充说明"
		if item, ok := m.questionList.SelectedItem().(questionItem); ok {
			if item.isSupplement {
				// 进入补充输入模式
				m.selectingSupplement = true
				m.supplementInput = ""
				return m, nil
			}
			// 普通选项 → 直接提交
			return m, m.sendAnswerCmd(item.text, false)
		}

	case "up", "down", "k", "j":
		if !m.selectingSupplement {
			var cmd tea.Cmd
			m.questionList, cmd = m.questionList.Update(msg)
			return m, cmd
		}

	default:
		if m.selectingSupplement {
			key := tea.Key(msg)
			switch key.Type {
			case tea.KeyRunes:
				// 可打印字符（含中文等多字节字符）
				m.supplementInput += string(key.Runes)
			case tea.KeyBackspace:
				// 回退键删除最后一个字符（正确处理 Unicode）
				if len(m.supplementInput) > 0 {
					runes := []rune(m.supplementInput)
					m.supplementInput = string(runes[:len(runes)-1])
				}
			case tea.KeySpace:
				m.supplementInput += " "
			}
		}
	}
	return m, nil
}

// ──────────────────────────── HTTP 回传 ────────────────────────────

// sendAnswerCmd 向服务端提交用户回答，异步执行。
func (m *Model) sendAnswerCmd(answer string, isSupplement bool) tea.Cmd {
	return func() tea.Msg {
		id := m.questionID
		err := m.client.Question.SendAnswer(id, answer, isSupplement)
		if err != nil {
			logger.Error("发送问题回答失败", logger.C(err), logger.S("id", id))
		}
		// 返回 nil，Update 中通过 tea.Batch 的回调处理界面退出
		return questionDoneMsg{answer: answer, isSupplement: isSupplement, err: err}
	}
}

// questionDoneMsg 问题回答完成消息
type questionDoneMsg struct {
	answer       string
	isSupplement bool
	err          error
}

// handleQuestionDone 处理回答完成，退出全屏问题界面。
func (m *Model) handleQuestionDone(msg questionDoneMsg) (tea.Model, tea.Cmd) {
	m.showingQuestionView = false
	m.questionID = ""
	m.questionContent = ""
	m.questionOptions = nil
	m.selectingSupplement = false
	m.supplementInput = ""

	// 追加 user 消息显示用户选择
	if msg.err != nil {
		m.messages = append(m.messages, message{role: "system", content: "回答提交失败: " + msg.err.Error()})
	} else {
		display := msg.answer
		if msg.isSupplement {
			display = "补充说明: " + msg.answer
		}
		m.messages = append(m.messages, message{role: "user", content: display})
	}

	m.needsAutoScroll = true
	return m, nil
}

// ──────────────────────────── 渲染 ────────────────────────────

// renderQuestionView 渲染全屏问题界面。
func (m *Model) renderQuestionView() string {
	var lines []string

	// 标题栏
	title := m.lip.SidebarActive.Render(" 需求澄清 — Esc 取消")
	sep := m.lip.SidebarSeparator.Render(strings.Repeat("─", m.width-4))
	lines = append(lines, title, sep, "")

	// 问题内容
	lines = append(lines, m.lip.User.Render("🤖 "+m.questionContent))
	lines = append(lines, "")

	// 选项列表
	lines = append(lines, m.questionList.View())

	// 补充输入模式
	if m.selectingSupplement {
		lines = append(lines, "")
		lines = append(lines, m.lip.SidebarActive.Render(" 请输入补充内容:"))
		lines = append(lines, m.supplementInput+"_")
		lines = append(lines, m.lip.SidebarSeparator.Render("  Enter 提交  Esc 返回选项"))
	}

	return strings.Join(lines, "\n")
}
