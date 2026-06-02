// Package confirm 提供工具调用确认机制的核心组件。
// 通过 Eino 中间件拦截工具调用，发送 SSE 事件给 TUI 等待用户确认，
// 支持 session 级别工具白名单和命令白名单持久化。
package confirm

import "time"

// ──────────────────────────── 核心数据类型 ────────────────────────────

// PendingEntry 单个待确认的工具调用项。
// ResultCh 缓冲为 1，确保 resolver 非阻塞写入后中间件可立即接收。
type PendingEntry struct {
	ID        string              // UUID，唯一标识
	ToolName  string              // 工具名称
	Arguments string              // 参数 JSON
	CallID    string              // Eino 工具调用 ID
	ResultCh  chan ConfirmResult  // 确认结果通道，缓冲 1
	CreatedAt time.Time           // 创建时间
	SessionID string              // 所属会话 ID
}

// ConfirmResult 工具确认结果，由 API handler 写入 channel。
type ConfirmResult struct {
	Approved bool   // true=确认执行，false=拒绝
	Action   string // "confirm" | "deny" | "allow" | "cleanup" | "timeout" | "canceled"
}

// ──────────────────────────── Actor 存储类型 ────────────────────────────

// Store 待确认存储 + session 级别工具白名单。
// 使用 Actor 模型：所有内部状态通过命令通道串行访问，无锁。
type Store struct {
	cmdCh chan func(s *storeState) // 命令通道，缓冲 256
	done  chan struct{}            // actor 退出信号
}

// storeState 存储内部状态，仅由 actor goroutine 访问。
// 对 map 的并发访问被 actor 的串行命令处理自然消除。
type storeState struct {
	pending        map[string]*PendingEntry    // confirmID → entry
	sessionAllowed map[string]map[string]bool  // sessionID → toolName → allowed
}

// ──────────────────────────── 中间件类型 ────────────────────────────

// ExecutorCallback executor 层向上传递事件的标准回调签名（与 aicallback 包一致）。
type ExecutorCallback func(event, content string) error

// ctxKey context key for executor callback.
type ctxKey struct{}

// sessionIDKey context key for session ID.
type sessionIDKey struct{}

// ConfirmEvent SSE tool_confirm 事件的 JSON 结构。
type ConfirmEvent struct {
	ID        string `json:"id"`         // 确认 UUID
	ToolName  string `json:"tool_name"`  // 工具名称
	Arguments string `json:"arguments"`  // 参数 JSON
}
