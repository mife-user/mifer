package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"mifer/pkg/errorer"

	"github.com/cloudwego/eino/schema"
)

type Memory struct {
	mu         sync.Mutex
	Messages   []*schema.Message
	savedCount int // 已持久化到文件的消息数量
	Cfg        MemCfg
}

type MemCfg struct {
	MemPath string
	Id      string
}

// ReplaceMessages 原子替换全部消息并全量重写持久化文件
func (m *Memory) ReplaceMessages(newMessages []*schema.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Messages = newMessages

	fileName := filepath.Join(m.Cfg.MemPath, fmt.Sprintf("%s.jsonl", m.Cfg.Id))
	f, err := os.Create(fileName)
	if err != nil {
		return errorer.NewS(errorer.ErrOpenFileFailed, err)
	}
	defer f.Close()

	for _, msg := range m.Messages {
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
