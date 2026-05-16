package tui

import (
	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
)

// ---- Bubble Tea 消息类型 ----

// message 对话消息（在消息列表中渲染的条目）
type message struct {
	role     string // "user" | "assistant" | "system"
	content  string // 原始文本内容
	rendered string // assistant 消息预渲染的 glamour ANSI 输出（其他角色为空）
}

// chatRespMsg SSE 流式响应全部积累完成后发出
type chatRespMsg struct {
	content string // 累积的完整 AI 响应
	err     error  // 网络或解析错误
}

// thinkingTickMsg 思考动画 500ms tick（自旋消息）
type thinkingTickMsg struct{}

// systemMsg 系统命令结果（/viewmemory、/excmem 等）
type systemMsg struct {
	content string
	err     error
}

// ---- TUI 核心 Model ----

// Model 持有 TUI 的全部状态，实现 bubbletea.Model 接口。
//
// 渲染管线（详见 view.go）：
//   View() → 构建消息行（带样式）→ scrollOff 切片 → 滚动指示器 → 填充 → JoinVertical(消息区, 输入区)
//
// 消息流（详见 update.go）：
//   用户输入 (tea.KeyMsg enter) → sendChatCmd (SSE 积累) → chatRespMsg → mark.Render() → 追加 message
type Model struct {
	// 依赖注入
	client *client.Client // HTTP API 客户端（Chat / Memory / Excmem）
	config *conf.Config   // 全局配置（样式、host、port）
	mark   *mark.Mark     // glamour markdown 渲染器（dark + notty 降级）
	lip    *lip.Style      // lipgloss 样式集合（前景色、分隔线等）

	// 消息与渲染
	messages    []message       // 对话消息列表（user / assistant / system）
	thinking    bool            // true 时 View 渲染旋转动画
	spinnerIdx  int             // 旋转动画当前帧索引 0..9
	err         string          // 当前错误文本（显示在消息区底部）
	lastMsgLine int             // 上次渲染时的消息总行数，用于检测新消息触发自动滚底

	// 视口滚动
	scrollOff int // 从顶部偏移的行数（0=底部，>0=向上滚动）
	width     int // 终端宽度（由 WindowSizeMsg 更新）
	height    int // 终端高度（由 WindowSizeMsg 更新）

	// 输入
	textarea     textarea.Model // bubbles 文本输入组件
	history      []string       // 历史输入记录（环形，最新追加）
	historyIdx   int            // 当前历史导航位置（-1 = 不在历史中）
	pendingInput string         // 进入历史导航前暂存的 textarea 内容（用于恢复）

	// Tab 补全
	completions   []string // 当前匹配到的命令列表
	completionIdx int      // 当前补全循环索引（-1 = 未激活）
	completionBase string  // 触发补全时的原始输入前缀（用于检测用户是否修改了输入）

	// 缓存：由 WindowSizeMsg 计算，避免 View 中重复计算
	contentHeight int // 消息区域可用行数
}
