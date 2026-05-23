package memory

import (
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

// GetCurrentID 返回当前记忆会话ID
func (m *Memory) GetCurrentID() string {
	return m.Cfg.Id
}

// SwitchSession 切换到新的记忆会话：先持久化当前会话的未保存消息，再从新会话的JSONL文件加载消息
func (m *Memory) SwitchSession(newID string) error {
	// 先持久化当前会话（Save 内部加锁）
	if err := m.Save(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Cfg.Id = newID
	msgs, err := load(&m.Cfg)
	if err != nil {
		return err
	}
	if msgs == nil {
		msgs = []*schema.Message{}
	}
	m.Messages = msgs
	m.savedCount = len(msgs)
	return nil
}

func Init(id string) (*Memory, error) {
	config := conf.GetConfig()
	var memory Memory
	memory.Cfg.Id = id
	memory.Cfg.MemPath = filepath.Join(config.Path.CfgPath, "/memory", filepath.Base(config.Path.Workdir))
	// 加载已有的对话历史
	msgs, err := load(&memory.Cfg)
	if err != nil {
		logger.Error("加载记忆文件失败", logger.C(err))
		return nil, err
	}
	if msgs == nil {
		msgs = []*schema.Message{}
	}
	memory.Messages = msgs
	memory.savedCount = len(msgs)
	return &memory, nil
}
