package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Save 将未持久化的新消息追加写入 JSONL 文件（每行一条 JSON）
func (m *Memory) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.Contains(m.Cfg.id, "..") || strings.Contains(m.Cfg.id, "/") || strings.Contains(m.Cfg.id, "\\") {
		return fmt.Errorf("id 包含非法字符: %s", m.Cfg.id)
	}
	if err := os.MkdirAll(m.Cfg.MemPath, 0755); err != nil {
		return fmt.Errorf("创建内存目录失败：%w", err)
	}

	newMsgs := m.Messages[m.savedCount:]
	if len(newMsgs) == 0 {
		return nil
	}

	fileName := filepath.Join(m.Cfg.MemPath, fmt.Sprintf("%s.jsonl", m.Cfg.id))
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败：%w", err)
	}
	defer f.Close()

	for _, msg := range newMsgs {
		line, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("序列化JSON失败：%w", err)
		}
		if _, err := f.Write(line); err != nil {
			return fmt.Errorf("写入文件失败：%w", err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			return fmt.Errorf("写入换行失败：%w", err)
		}
	}
	m.savedCount = len(m.Messages)
	return nil
}
