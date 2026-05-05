package memory

import (
	"mifer/pkg/conf"

	"github.com/cloudwego/eino/schema"
)

type Memory struct {
	Messages []*schema.Message
}

func Init(config *conf.Config, id []byte) *Memory {
	var memory Memory
	// 加载已有的对话历史
	msgs, err := load(config, id)
	if err != nil {
		return nil
	}
	memory.Messages = msgs
	return &memory
}
