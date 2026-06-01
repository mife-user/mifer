package tui

// ============================================================================
// type.go — 类型定义
// ============================================================================
//
// 本文件定义 TUI（终端用户界面）所需的所有数据结构，包括：
//   1. Bubble Tea 消息类型 —— 在 Update 方法中通过类型断言分发
//   2. Model 结构体 —— 持有全部 UI 状态，实现 bubbletea.Model 接口
//
// Bubble Tea 框架基于 Elm Architecture 设计，核心是三个方法：
//   Init()   → 返回初始命令（如光标闪烁）
//   Update() → 接收消息，更新状态，返回新命令
//   View()   → 根据当前状态渲染 UI 字符串
//
// 消息流转过程：
//   用户按键 → tea.KeyMsg → Model.Update()
//   子组件 tick → spinner.TickMsg → Model.Update()
//   HTTP 响应 → chatRespMsg/systemMsg → Model.Update()
//   窗口变化 → tea.WindowSizeMsg → Model.Update()
//   鼠标事件 → tea.MouseMsg → viewport.Update() (委托)

import (
	"context"
	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// chatRespMsg 由 sendChatCmd 在 SSE 流全部积累完成后发出。
//
// 异步流程：
//
//	handleEnter() → tea.Batch(sendChatCmd, spinner.Tick)
//	sendChatCmd 在 goroutine 中通过 HTTP SSE 积累 AI 响应
//	积累完成后返回 chatRespMsg 到 Update()
//
// 与 systemMsg 的区别：chatRespMsg 走 markdown 渲染 + 追加 assistant 消息，
// systemMsg 直接追加原始内容为 system 消息。
type chatRespMsg struct {
	content string // 累积的完整 AI 响应文本
	err     error  // 网络或解析错误（非 nil 时显示错误不追加消息）
}

// ============================================================================
// Model — TUI 核心状态
// ============================================================================

// Model 持有 TUI 的全部状态，实现 bubbletea.Model 接口。
//
// 字段分组说明：
//
// 【依赖注入】— 由外部传入，TUI 生命周期内不变
//
//	client   — HTTP API 客户端，用于发送聊天请求、加载记忆、切换会话
//	config   — 全局配置（样式颜色、历史条数、命令列表等）
//	mark     — glamour markdown 渲染器（dark 主题 + notty 降级）
//	lip      — lipgloss 样式集合（前景色、分隔线等），在 View() 中使用
//
// 【消息与渲染】— 对话内容与视觉状态
//
//	messages        — 对话消息列表（user / assistant / system）
//	thinking        — 是否正在等待 AI 响应（控制 spinner 动画显示与驱动）
//	spinner         — bubbles/spinner 组件，管理旋转动画帧
//	err             — 当前错误文本（显示在消息区底部，非空时显示红色）
//	needsAutoScroll — 新消息到达标记，在 View() 中触发 viewport.GotoBottom()
//
// 【视口】— 可滚动的消息显示区
//
//	viewport        — bubbles/viewport 组件，处理滚动、鼠标滚轮、内容裁剪
//	width           — 终端宽度（列数）
//	height          — 终端高度（行数）
//	contentHeight   — 消息区域可用行数（终端高度 - textarea 高度 - 间距）
//
// 【输入】— 用户交互
//
//	textarea        — bubbles/textarea 组件，多行文本输入
//	history         — 历史输入记录的环形缓冲区（最新在末尾）
//	historyIdx      — 当前历史导航位置（-1 表示不在历史中）
//	pendingInput    — 进入历史导航前暂存的 textarea 内容，用于恢复
//
// 【Tab 补全】— 命令自动完成
//
//	completions     — 当前匹配到的命令列表
//	completionIdx   — 当前补全循环索引（-1 表示未激活补全）
//	completionBase  — 触发补全时的输入前缀，用于检测用户是否修改了输入
//
// 完整消息流：
//
//	用户输入 → KeyMsg(enter) → handleEnter()
//	  → 追加 user message → 设置 thinking=true → 启动 spinner
//	  → tea.Batch(sendChatCmd, spinner.Tick)
//	    sendChatCmd → HTTP SSE 积累 → chatRespMsg
//	    spinner.Tick → TickMsg → 帧推进 → 自旋
//	  → chatRespMsg → 设置 thinking=false → mark.Render() → 追加 assistant message
//	  → View() 渲染所有消息到 viewport → 显示到终端
type Model struct {
	// 依赖注入
	client *client.Client // HTTP API 客户端（Chat / Memory / Excmem）
	mark   *mark.Mark     // glamour markdown 渲染器（dark + notty 降级）
	lip    *lip.Style     // lipgloss 样式集合（前景色、分隔线等）

	// 消息与渲染
	messages        []message     // 对话消息列表（user / assistant / system）
	thinking        bool          // true 时 View() 渲染 spinner 动画
	spinner         spinner.Model // bubbles/spinner 旋转动画组件
	err             string        // 当前错误文本（显示在消息区底部）
	needsAutoScroll bool          // 新消息到达标记，触发 viewport.GotoBottom()

	// 视口
	viewport viewport.Model // bubbles/viewport 滚动视口组件
	width    int            // 终端宽度（由 WindowSizeMsg 更新）
	height   int            // 终端高度（由 WindowSizeMsg 更新）

	// 输入
	textarea     textarea.Model // bubbles/textarea 文本输入组件
	history      []string       // 历史输入记录（环形，最新追加）
	historyIdx   int            // 当前历史导航位置（-1 = 不在历史中）
	pendingInput string         // 进入历史导航前暂存的 textarea 内容（用于恢复）

	// Tab 补全
	completions         []string          // 当前匹配到的命令列表
	completionDescs     map[string]string // 命令 → 描述映射（用于补全列表渲染）
	completionIdx       int               // 当前补全循环索引（-1 = 未激活）
	completionBase      string            // 触发补全时的原始输入前缀
	showingCompletions  bool              // 是否显示补全列表（输入 / 开头且有匹配）

	// 缓存：由 WindowSizeMsg 计算，避免 View 中重复计算
	contentHeight int // 消息区域可用行数

	// 侧边栏
	sidebar   SidebarState   // agent/工具状态跟踪（Current + Log + Token）
	sidebarVP viewport.Model // 状态日志滚动视口

	// 流式传输
	cancel            context.CancelFunc // 用于取消正在进行的 SSE 流（非 nil 表示可取消）
	streamCh          chan tea.Msg     // 流式消息通道（非nil时表示正在流式传输中）
	accBuf            *strings.Builder // 累积AI响应内容的缓冲区
	agentContentStart int              // accBuf 中当前 agent 内容的起始字节偏移

	// 记忆选择模式
	selectingMem  bool       // 是否正在显示记忆选择列表
	pendingMemCmd string     // 等待执行的命令（"/viewmemory" 或 "/excmem"）
	memoryList    list.Model // bubbles/list 记忆选择组件

	// 全屏记忆查看模式
	showingMemoryView bool           // 是否处于全屏记忆查看模式
	memoryViewContent string         // 记忆内容（原始文本）
	memoryViewport    viewport.Model // 独立的 viewport 用于全屏记忆查看

	// 回退选择模式
	selectingReback bool       // 是否正在显示回退选择列表
	rebackList      list.Model // bubbles/list 回退选择组件

	// 计划选择模式
	selectingPlan bool       // 是否正在显示计划选择列表
	planList      list.Model // bubbles/list 计划选择组件

	// 全屏计划查看模式
	showingPlanView bool           // 是否处于全屏计划查看模式
	planViewContent string         // 计划文件内容
	planViewport    viewport.Model // 独立的 viewport 用于全屏计划查看

	// 工具确认
	selectingConfirm bool                  // 是否处于确认选择模式
	confirmQueue     []*ToolConfirmPrompt  // 待确认队列
	currentConfirm   *ToolConfirmPrompt    // 当前正在确认的项（队首）
	confirmList      list.Model            // bubbles/list 选择组件 [Yes | No | Allow]
	sessionAllowed   map[string]bool       // 当前会话已 Allow 的工具名集合
}

// ============================================================================
// Bubble Tea 消息类型
// ============================================================================
// 所有消息类型都实现了 tea.Msg 接口（空接口），在 Update() 中通过类型断言分发。

// message 对话消息，表示消息列表中渲染的一条记录。
//
// role 决定 View() 中的渲染方式：
//
//	"user"      → 绿色 "You: " 前缀，整行渲染
//	"assistant" → 优先使用 glamour 预渲染的 ANSI 输出
//	"system"    → 青色整行渲染，支持多行
type message struct {
	role     string // "user" | "assistant" | "system"
	content  string // 原始文本内容
	rendered string // assistant 消息预渲染的 glamour ANSI 输出（其他角色为空）
}
