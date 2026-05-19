package tui

// ============================================================================
// memory.go — 记忆会话类型与逻辑
// ============================================================================
//
// 记忆管理支持两个操作：
//   /viewmemory [id] — 查看记忆（无 id 进入选择列表，有 id 直接查看）
//   /excmem <id>     — 切换当前会话（无 id 进入选择列表）
//
// 流程：
//   1. handleEnter() 识别命令 → listMemoriesCmd 异步获取记忆列表
//   2. memoryListMsg → handleMemoryList 校验 ID 或进入选择模式
//   3. 选择模式下 Enter → handleMemorySelect → loadMemoryCmd / excmemCmd
//   4. memoryViewMsg → handleMemoryView 进入全屏记忆查看模式

import (
	"mifer/cli/client"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// memoryListMsg 异步获取记忆列表的结果，由 listMemoriesCmd 发出。
//
// 携带触发上下文（cmd 和 argID），用于判断是展示选择列表还是直接执行命令。
type memoryListMsg struct {
	current string   // 当前记忆ID
	ids     []string // 所有可用记忆ID列表
	err     error    // 网络或解析错误
	cmd     string   // 触发命令："/viewmemory" 或 "/excmem"
	argID   string   // 命令后跟的ID（空表示无ID，需进入选择模式）
}

// memoryViewMsg /viewmemory 加载完成，进入全屏记忆查看模式
type memoryViewMsg struct {
	content string // 格式化的对话记忆文本
	err     error
}

// ============================================================================
// memoryItem — list.Item 实现，用于记忆选择列表
// ============================================================================

// memoryItem 实现 bubbles/list.Item 接口，表示一条可选的记忆会话。
type memoryItem struct {
	id      string // 记忆ID
	current bool   // 是否为当前会话
}

func (i memoryItem) Title() string { return i.id }
func (i memoryItem) Description() string {
	if i.current {
		return "(当前)"
	}
	return ""
}
func (i memoryItem) FilterValue() string { return i.id }

// ============================================================================
// Update() 中的记忆消息处理器
// ============================================================================

// handleMemoryList 处理记忆列表结果 → 校验ID或进入选择模式
func (m *Model) handleMemoryList(msg memoryListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = "错误: " + msg.err.Error()
		return m, nil
	}

	// 带ID参数：校验后直接执行
	if msg.argID != "" {
		found := false
		for _, id := range msg.ids {
			if id == msg.argID {
				found = true
				break
			}
		}
		if !found {
			m.err = "记忆ID不存在: " + msg.argID
			return m, nil
		}
		// ID存在，直接执行对应命令
		switch msg.cmd {
		case "/viewmemory":
			return m, loadMemoryCmd(m.client, msg.argID)
		default:
			return m, excmemCmd(m.client, msg.argID)
		}
	}

	// 无ID参数：进入选择模式，显示记忆列表
	var items []list.Item
	for _, id := range msg.ids {
		items = append(items, memoryItem{id: id, current: id == msg.current})
	}
	m.selectingMem = true
	m.pendingMemCmd = msg.cmd
	m.memoryList.SetWidth(m.width / 4) // 初始宽度，View中会重新设置
	return m, m.memoryList.SetItems(items)
}

// handleMemoryView 处理记忆查看结果 → 进入全屏记忆查看模式
func (m *Model) handleMemoryView(msg memoryViewMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = "错误: " + msg.err.Error()
		return m, nil
	}
	m.showingMemoryView = true
	m.memoryViewContent = msg.content
	m.memoryViewport.SetContent(msg.content)
	m.memoryViewport.GotoTop()
	return m, nil
}

// handleMemorySelect 处理记忆选择列表中的 Enter 键
//
// 获取当前选中的记忆ID，根据 pendingMemCmd 执行对应命令后退出选择模式。
func (m *Model) handleMemorySelect() (tea.Model, tea.Cmd) {
	item := m.memoryList.SelectedItem()
	if item == nil {
		m.selectingMem = false
		m.pendingMemCmd = ""
		return m, nil
	}
	mi := item.(memoryItem)
	cmd := m.pendingMemCmd
	m.selectingMem = false
	m.pendingMemCmd = ""
	if cmd == "/viewmemory" {
		return m, loadMemoryCmd(m.client, mi.id)
	}
	return m, excmemCmd(m.client, mi.id)
}

// ============================================================================
// 记忆操作命令
// ============================================================================

// loadMemoryCmd 异步加载指定会话的对话记忆
//
// 通过 HTTP GET /api/memory/:id 获取 JSONL 格式的对话历史。
// id 为空时默认加载 "default" 会话。
// 返回 memoryViewMsg 触发全屏记忆查看模式。
func loadMemoryCmd(client *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if id == "" {
			id = "default"
		}
		memory, err := client.Memory.Load(id)
		if err != nil {
			return memoryViewMsg{err: err}
		}
		if memory == "" {
			memory = "(暂无对话记忆)"
		}
		return memoryViewMsg{content: memory}
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

// listMemoriesCmd 异步获取记忆列表，用于校验ID或进入选择模式
//
// 由 handleEnter() 中 /viewmemory 和 /excmem 命令触发。
// 结果以 memoryListMsg 返回，携带触发上下文（命令名和参数ID）。
func listMemoriesCmd(client *client.Client, cmd, id string) tea.Cmd {
	return func() tea.Msg {
		current, ids, err := client.Memory.List()
		return memoryListMsg{current: current, ids: ids, err: err, cmd: cmd, argID: id}
	}
}
