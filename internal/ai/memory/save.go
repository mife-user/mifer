package memory

import (
	"encoding/json"
	"fmt"
	"mifer/pkg/errorer"
	"os"
	"path/filepath"
	"strings"
)

// Save 将未持久化的新消息追加写入 JSONL 文件（每行一条 JSON）
func (m *Memory) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.Contains(m.Cfg.Id, "..") || strings.Contains(m.Cfg.Id, "/") || strings.Contains(m.Cfg.Id, "\\") {
		return errorer.NewF(errorer.ErrIDIllegalChars, m.Cfg.Id)
	}
	if err := os.MkdirAll(m.Cfg.MemPath, 0755); err != nil {
		return errorer.NewS(errorer.ErrCreateMemoryDirFailed, err)
	}

	newMsgs := m.Messages[m.savedCount:]
	if len(newMsgs) == 0 {
		return nil
	}

	fileName := filepath.Join(m.Cfg.MemPath, fmt.Sprintf("%s.jsonl", m.Cfg.Id))
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errorer.NewS(errorer.ErrOpenFileFailed, err)
	}
	defer f.Close()

	for _, msg := range newMsgs {
		line, err := json.Marshal(msg)
		if err != nil {
			return errorer.NewS(errorer.ErrSerializeJSONFailed, err)
		}
		if _, err := f.Write(line); err != nil {
			return errorer.NewS(errorer.ErrWriteFileFailed, err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			return errorer.NewS(errorer.ErrWriteNewlineFailed, err)
		}
	}
	m.savedCount = len(m.Messages)
	return nil
}
