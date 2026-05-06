package memory

import (
	"fmt"
	"mifer/pkg/conf"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

type Memory struct {
	Messages []*schema.Message
	Cfg      MemCfg
}

type MemCfg struct {
	MemPath string
	id      []byte
}

func Init(config *conf.Config, id []byte) (*Memory, error) {
	var memory Memory
	memory.Cfg.id = id
	// 确定记忆文件存储路径
	if config.Env == "dev" {
		memory.Cfg.MemPath = filepath.Join("./memory", filepath.Base(config.Workdir))
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败：%w", err)
		}
		memory.Cfg.MemPath = filepath.Join(home, "/mifer/memory", filepath.Base(config.Workdir))
	}
	// 加载已有的对话历史
	msgs, err := load(&memory.Cfg)
	if err != nil {
		return nil, err
	}
	memory.Messages = msgs
	return &memory, nil
}
