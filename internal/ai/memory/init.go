package memory

import (
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"path/filepath"
	"sync"

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

func Init(config *conf.Config, id string) (*Memory, error) {
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
