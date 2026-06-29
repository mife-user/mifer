package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

type Memory struct {
	mu         sync.Mutex
	messages   []*schema.Message // 未导出，外部必须通过 Messages() 访问以防止 data race
	savedCount int               // 已持久化到文件的消息数量
	Cfg        MemCfg
}

// Messages 返回当前记忆消息切片的只读引用（持有锁）。
// 调用方不得修改返回的切片内容；所有修改操作必须通过 AppendUser/AppendAssistant 等方法。
func (m *Memory) Messages() []*schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages
}

// Len 返回当前记忆中的消息数量（持有锁）。
func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

type MemCfg struct {
	MemPath string
	Id      string
}

// ReplaceMessages 原子替换全部消息并全量重写持久化文件
func (m *Memory) ReplaceMessages(newMessages []*schema.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = newMessages

	fileName := filepath.Join(m.Cfg.MemPath, fmt.Sprintf("%s.jsonl", m.Cfg.Id))
	f, err := os.Create(fileName)
	if err != nil {
		logger.Error("创建记忆文件失败", logger.C(err))
		return errorer.NewS(errorer.ErrOpenFileFailed, err)
	}
	defer f.Close()

	for _, msg := range m.messages {
		line, err := json.Marshal(msg)
		if err != nil {
			logger.Error("序列化记忆失败", logger.C(err))
			return errorer.NewS(errorer.ErrSerializeJSONFailed, err)
		}
		if _, err := f.Write(line); err != nil {
			logger.Error("写入记忆失败", logger.C(err))
			return errorer.NewS(errorer.ErrWriteFileFailed, err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			logger.Error("写入记忆换行符失败", logger.C(err))
			return errorer.NewS(errorer.ErrWriteNewlineFailed, err)
		}
	}
	m.savedCount = len(m.messages)
	return nil
}
