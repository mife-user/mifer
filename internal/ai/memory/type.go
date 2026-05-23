package memory

import (
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
