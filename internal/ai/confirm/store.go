// Package confirm 提供工具调用确认机制的核心组件。
// 通过 Eino 中间件拦截工具调用，发送 SSE 事件给 TUI 等待用户确认，
// 支持 session 级别工具白名单和命令白名单持久化。
package confirm

import (
	"sync"
	"time"

	"mifer/pkg/logger"
)

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
	Action   string // "confirm" | "deny" | "allow"
}

// Store 待确认存储 + session 级别工具白名单。
// 线程安全，通过 sync.RWMutex 保护。
type Store struct {
	mu             sync.RWMutex
	pending        map[string]*PendingEntry          // confirmID → entry
	sessionAllowed map[string]map[string]bool        // sessionID → toolName → allowed
}

// NewStore 创建空存储实例并启动超时清理协程。
func NewStore() *Store {
	s := &Store{
		pending:        make(map[string]*PendingEntry),
		sessionAllowed: make(map[string]map[string]bool),
	}
	go s.cleanupLoop(30 * time.Second)
	return s
}

// Add 将待确认项加入存储。
func (s *Store) Add(entry *PendingEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[entry.ID] = entry
}

// Get 通过 ID 查找待确认项。
func (s *Store) Get(id string) (*PendingEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.pending[id]
	return e, ok
}

// Remove 从存储中移除指定待确认项（不关闭 channel）。
func (s *Store) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, id)
}

// Resolve 解析指定确认项：写入结果到 channel 并从存储移除。
// 若 ID 不存在则静默忽略（可能已被超时清理）。
func (s *Store) Resolve(id string, result ConfirmResult) {
	s.mu.Lock()
	entry, ok := s.pending[id]
	if ok {
		delete(s.pending, id)
	}
	s.mu.Unlock()

	if ok {
		// 非阻塞写入（channel 缓冲为 1）
		select {
		case entry.ResultCh <- result:
		default:
			logger.Warn("确认结果通道已满，可能已超时",
				logger.S("id", id), logger.S("action", result.Action))
		}
	}
}

// AllowTool 将指定工具加入 session 级别白名单。
func (s *Store) AllowTool(sessionID, toolName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionAllowed[sessionID] == nil {
		s.sessionAllowed[sessionID] = make(map[string]bool)
	}
	s.sessionAllowed[sessionID][toolName] = true
}

// IsAllowed 检查指定 session 下某工具是否已被 Allow。
func (s *Store) IsAllowed(sessionID, toolName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.sessionAllowed[sessionID]; ok {
		return m[toolName]
	}
	return false
}

// Cleanup 清理指定 session 的所有待确认项和 session 白名单。
// 关闭所有待确认 channel，避免 goroutine 泄漏。
func (s *Store) Cleanup(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 关闭该 session 的所有待确认 channel
	for id, entry := range s.pending {
		if entry.SessionID == sessionID {
			// 发送拒绝结果，解除中间件阻塞
			select {
			case entry.ResultCh <- ConfirmResult{Approved: false, Action: "cleanup"}:
			default:
			}
			delete(s.pending, id)
		}
	}
	// 清除 session 白名单
	delete(s.sessionAllowed, sessionID)
}

// PendingCount 返回当前待确认项数量（用于监控）。
func (s *Store) PendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pending)
}

// cleanupLoop 定期清理超时的待确认项。
func (s *Store) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, entry := range s.pending {
			// 超过 5 分钟未处理的清理（超时时间是 60s，5 分钟是保守估计）
			if now.Sub(entry.CreatedAt) > 5*time.Minute {
				select {
				case entry.ResultCh <- ConfirmResult{Approved: false, Action: "timeout"}:
				default:
				}
				delete(s.pending, id)
				logger.Debug("清理超时确认项", logger.S("id", id),
					logger.S("tool", entry.ToolName))
			}
		}
		s.mu.Unlock()
	}
}
