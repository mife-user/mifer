package memory

import (
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"os"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/schema"
)

const maxIDLen = 50

// Rename 重命名当前会话记忆文件。若文件尚未创建则仅修改内存中的 ID；若文件已存在则同时重命名 .jsonl 和 _snapshots/ 目录。
func (m *Memory) Rename(newID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newID = strings.TrimSpace(newID)
	if strings.ContainsAny(newID, "/\\") {
		return errorer.NewF(errorer.ErrIDIllegalChars, newID)
	}

	return m.renameLocked(newID)
}

// AutoRenameFromFirstMessage 取首条用户消息前缀作为新会话名称，静默失败。
func (m *Memory) AutoRenameFromFirstMessage() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.messages {
		if msg.Role == schema.User {
			newName := sanitizeAutoName(msg.Content, 20)
			if newName == "" || newName == m.Cfg.Id {
				return nil
			}
			if err := m.renameLocked(newName); err != nil {
				logger.Warn("自动重命名失败", logger.C(err))
				return err
			}
			logger.Info("会话自动重命名", logger.S("new_name", newName))
			return nil
		}
	}
	return nil
}

// renameLocked 执行实际的重命名逻辑，调用方必须持有 m.mu 锁。
func (m *Memory) renameLocked(newID string) error {
	if err := validateID(newID); err != nil {
		return err
	}
	if len([]rune(newID)) > maxIDLen {
		return errorer.NewF(errorer.ErrIDTooLong, maxIDLen)
	}

	// 冲突检查
	ids, err := m.listIDsLocked()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == newID && id != m.Cfg.Id {
			return errorer.NewF(errorer.ErrIDConflict, newID)
		}
	}

	oldID := m.Cfg.Id
	oldPath, err := buildFilePath(m.Cfg.MemPath, oldID)
	if err != nil {
		return err
	}

	// 旧文件不存在 → 仅修改内存 ID（文件将在下次 Save 时创建）
	if _, statErr := os.Stat(oldPath); os.IsNotExist(statErr) {
		m.Cfg.Id = newID
		m.fileCreated = false
		return nil
	}

	// 重命名 .jsonl 文件
	newPath, err := buildFilePath(m.Cfg.MemPath, newID)
	if err != nil {
		return err
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		logger.Error("重命名记忆文件失败", logger.C(err), logger.S("old", oldPath), logger.S("new", newPath))
		return err
	}

	// 重命名快照目录（若存在）
	oldSnapDir := oldID + "_snapshots"
	newSnapDir := newID + "_snapshots"
	oldSnapPath := strings.Replace(oldPath, oldID+".jsonl", oldSnapDir, 1)
	newSnapPath := strings.Replace(newPath, newID+".jsonl", newSnapDir, 1)
	if _, err := os.Stat(oldSnapPath); err == nil {
		if err := os.Rename(oldSnapPath, newSnapPath); err != nil {
			logger.Warn("重命名快照目录失败，已回滚记忆文件重命名", logger.C(err))
			// 回滚 .jsonl 重命名
			_ = os.Rename(newPath, oldPath)
			return err
		}
	}

	m.Cfg.Id = newID
	m.fileCreated = false
	return nil
}

// listIDsLocked 与 ListIDs 逻辑相同但不加锁（调用方已持有锁时使用）。
func (m *Memory) listIDsLocked() ([]string, error) {
	return (&Memory{Cfg: m.Cfg}).ListIDs()
}

// sanitizeAutoName 从消息内容中提取合法 ID 前缀。
// 保留字母、数字、中文、下划线，其余替换为 '_'，合并连续下划线。
func sanitizeAutoName(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) > maxLen {
		runes = runes[:maxLen]
	}

	var b strings.Builder
	b.Grow(len(runes))
	lastUnderscore := false
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			if r == '_' {
				if lastUnderscore {
					continue
				}
				lastUnderscore = true
			} else {
				lastUnderscore = false
			}
			b.WriteRune(r)
		} else {
			if !lastUnderscore && b.Len() > 0 {
				lastUnderscore = true
				b.WriteRune('_')
			}
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "会话"
	}
	return result
}
