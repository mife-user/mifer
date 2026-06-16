package question

import (
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
		pending: make(map[string]*QuestionEntry),
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
				return
			}
			cmd(state)

		case now := <-ticker.C:
			// 定期清理超时的待回答问题（超过 5 分钟未处理的保守清理）
			for id, entry := range state.pending {
				if now.Sub(entry.CreatedAt) > 5*time.Minute {
					select {
					case entry.ResultCh <- QuestionResult{Answer: "", IsSupplement: false}:
					default:
					}
					delete(state.pending, id)
					logger.Debug("清理超时问题项", logger.S("id", id))
				}
			}
		}
	}
}

// ──────────────────────────── 公开方法 ────────────────────────────

// Add 将待回答问题加入存储。
func (s *Store) Add(entry *QuestionEntry) {
	s.cmdCh <- func(state *storeState) {
		state.pending[entry.ID] = entry
	}
}

// Get 通过 ID 查找待回答问题。
func (s *Store) Get(id string) (*QuestionEntry, bool) {
	type resp struct {
		entry *QuestionEntry
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

// Remove 从存储中移除指定待回答问题（不关闭 channel）。
func (s *Store) Remove(id string) {
	s.cmdCh <- func(state *storeState) {
		delete(state.pending, id)
	}
}

// Resolve 解析指定问题：写入结果到 channel 并从存储移除。
// 若 ID 不存在则静默忽略（可能已被超时清理）。
func (s *Store) Resolve(id string, result QuestionResult) {
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
				logger.Warn("问题结果通道已满，可能已超时",
					logger.S("id", id))
			}
		}
	}
}

// Cleanup 清理指定 session 的所有待回答问题。
// 关闭所有待回答 channel，避免 goroutine 泄漏。
func (s *Store) Cleanup(sessionID string) {
	s.cmdCh <- func(state *storeState) {
		for id, entry := range state.pending {
			if entry.SessionID == sessionID {
				select {
				case entry.ResultCh <- QuestionResult{Answer: "", IsSupplement: false}:
				default:
				}
				delete(state.pending, id)
			}
		}
	}
}

// PendingCount 返回当前待回答问题数量（用于监控）。
func (s *Store) PendingCount() int {
	respCh := make(chan int, 1)
	s.cmdCh <- func(state *storeState) {
		respCh <- len(state.pending)
	}
	return <-respCh
}

// Close 优雅关闭 Store 的 actor goroutine。
// 调用后不应再使用 Store。
func (s *Store) Close() {
	s.cmdCh <- nil // 发送停止信号
	<-s.done       // 等待 actor 退出
}
