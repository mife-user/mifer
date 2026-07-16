package confirm

import (
	"context"
	"time"

	"mifer/pkg/logger"
)

const cmdChBufSize = 256 // 命令通道缓冲区大小

// NewStore 创建空存储实例并启动 actor goroutine。
func NewStore() *Store {
	s := &Store{
		cmdCh: make(chan func(s *storeState), cmdChBufSize),
		done:  make(chan struct{}),
	}
	state := &storeState{
		pending:        make(map[string]*PendingEntry),
		sessionAllowed: make(map[string]map[string]bool),
	}
	go s.runActor(state)
	return s
}

// ──────────────────────────── Actor 主循环 ────────────────────────────

// runActor Store 的 actor goroutine，串行处理所有命令和定期清理。
func (s *Store) runActor(state *storeState) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case cmd := <-s.cmdCh:
			if cmd == nil {
				// nil 命令为停止信号（由 Close() 发送）
				close(s.done)
				return
			}
			cmd(state)

		case now := <-ticker.C:
			// 定期清理超时的待确认项（超过 5 分钟未处理的保守清理）
			for id, entry := range state.pending {
				if now.Sub(entry.CreatedAt) > 5*time.Minute {
					select {
					case entry.ResultCh <- ConfirmResult{Approved: false, Action: "timeout"}:
					default:
					}
					delete(state.pending, id)
					logger.Debug(context.Background(), "清理超时确认项", logger.S("id", id),
						logger.S("tool", entry.ToolName))
				}
			}
		}
	}
}

// ──────────────────────────── 公开方法 ────────────────────────────

// Add 将待确认项加入存储。
func (s *Store) Add(entry *PendingEntry) {
	s.cmdCh <- func(state *storeState) {
		state.pending[entry.ID] = entry
	}
}

// Get 通过 ID 查找待确认项。
func (s *Store) Get(id string) (*PendingEntry, bool) {
	type resp struct {
		entry *PendingEntry
		ok    bool
	}
	respCh := make(chan resp, 1)
	s.cmdCh <- func(state *storeState) {
		e, ok := state.pending[id]
		respCh <- resp{e, ok}
	}
	r := <-respCh
	return r.entry, r.ok
}

// Remove 从存储中移除指定待确认项（不关闭 channel）。
func (s *Store) Remove(id string) {
	s.cmdCh <- func(state *storeState) {
		delete(state.pending, id)
	}
}

// Resolve 解析指定确认项：写入结果到 channel 并从存储移除。
// 若 ID 不存在则静默忽略（可能已被超时清理）。
func (s *Store) Resolve(id string, result ConfirmResult) {
	s.cmdCh <- func(state *storeState) {
		entry, ok := state.pending[id]
		if ok {
			delete(state.pending, id)
		}
		if ok {
			// 非阻塞写入（channel 缓冲为 1）
			select {
			case entry.ResultCh <- result:
			default:
				logger.Warn(context.Background(), "确认结果通道已满，可能已超时",
					logger.S("id", id), logger.S("action", result.Action))
			}
		}
	}
}

// AllowTool 将指定工具加入 session 级别白名单。
func (s *Store) AllowTool(sessionID, toolName string) {
	s.cmdCh <- func(state *storeState) {
		if state.sessionAllowed[sessionID] == nil {
			state.sessionAllowed[sessionID] = make(map[string]bool)
		}
		state.sessionAllowed[sessionID][toolName] = true
	}
}

// IsAllowed 检查指定 session 下某工具是否已被 Allow。
func (s *Store) IsAllowed(sessionID, toolName string) bool {
	respCh := make(chan bool, 1)
	s.cmdCh <- func(state *storeState) {
		if m, ok := state.sessionAllowed[sessionID]; ok {
			respCh <- m[toolName]
			return
		}
		respCh <- false
	}
	return <-respCh
}

// Cleanup 清理指定 session 的所有待确认项和 session 白名单。
// 关闭所有待确认 channel，避免 goroutine 泄漏。
func (s *Store) Cleanup(sessionID string) {
	s.cmdCh <- func(state *storeState) {
		for id, entry := range state.pending {
			if entry.SessionID == sessionID {
				select {
				case entry.ResultCh <- ConfirmResult{Approved: false, Action: "cleanup"}:
				default:
				}
				delete(state.pending, id)
			}
		}
		delete(state.sessionAllowed, sessionID)
	}
}

// PendingCount 返回当前待确认项数量（用于监控）。
func (s *Store) PendingCount() int {
	respCh := make(chan int, 1)
	s.cmdCh <- func(state *storeState) {
		respCh <- len(state.pending)
	}
	return <-respCh
}

// Close 优雅关闭 Store 的 actor goroutine。
// 调用后不应再使用 Store。预留用于未来资源清理。
func (s *Store) Close() {
	s.cmdCh <- nil // 发送停止信号
	<-s.done       // 等待 actor 退出
}
