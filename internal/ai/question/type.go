// Package question 提供需求澄清问答机制的核心组件。
// 当 AI 需要明确用户需求时，通过 ask_user 工具阻塞等待用户选择或补充答案，
// 通过 SSE 事件发送问题到 TUI，用户回答后通过 HTTP 回传解除阻塞。
package question

import "time"

// ──────────────────────────── 核心数据类型 ────────────────────────────

// QuestionEntry 单个待回答的问题项。
// ResultCh 缓冲为 1，确保 resolver 非阻塞写入后阻塞方可立即接收。
type QuestionEntry struct {
	ID        string            // UUID，唯一标识
	Question  string            // 问题文本
	Options   []string          // 答案选项（不含"补充"，TUI 自动追加）
	ResultCh  chan QuestionResult // 回答结果通道，缓冲 1
	CreatedAt time.Time         // 创建时间
	SessionID string            // 所属会话 ID
}

// QuestionResult 用户回答结果，由 HTTP handler 写入 channel。
type QuestionResult struct {
	Answer       string // 用户选择的答案或补充内容
	IsSupplement bool   // 是否为补充输入
}

// ──────────────────────────── Actor 存储类型 ────────────────────────────

// Store 待回答问题存储。
// 使用 Actor 模型：所有内部状态通过命令通道串行访问，无锁。
type Store struct {
	cmdCh chan func(s *storeState) // 命令通道，缓冲 256
	done  chan struct{}            // actor 退出信号
}

// storeState 存储内部状态，仅由 actor goroutine 访问。
type storeState struct {
	pending map[string]*QuestionEntry // questionID → entry
}

// ──────────────────────────── Context Key 类型 ────────────────────────────

// ExecutorCallback executor 层向上传递事件的标准回调签名（与 aicallback 包一致）。
type ExecutorCallback func(event, content string) error

// ctxKey context key for executor callback.
type ctxKey struct{}

// storeKey context key for question store.
type storeKey struct{}

// sessionIDKey context key for session ID.
type sessionIDKey struct{}

// ──────────────────────────── SSE 事件类型 ────────────────────────────

// AskUserEvent SSE ask_user 事件的 JSON 结构。
type AskUserEvent struct {
	ID       string   `json:"id"`       // 问题 UUID
	Question string   `json:"question"` // 问题文本
	Options  []string `json:"options"`  // 答案选项
}
