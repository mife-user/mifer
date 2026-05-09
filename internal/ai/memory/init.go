package memory

import (
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudwego/eino/schema"
)

type Memory struct {
	mu       sync.Mutex
	Messages []*schema.Message
	Cfg      MemCfg
}

type MemCfg struct {
	MemPath string
	id      string
}

func Init(config *conf.Config, id string) (*Memory, error) {
	var memory Memory
	memory.Cfg.id = id
	// 确定记忆文件存储路径
	if config.Env == "dev" {
		absPath, err := filepath.Abs("./memory")
		if err != nil {
			logger.Error("获取记忆目录绝对路径失败", logger.C(err))
			return nil, fmt.Errorf("获取记忆目录绝对路径失败：%w", err)
		}
		memory.Cfg.MemPath = filepath.Join(absPath, filepath.Base(config.Workdir))
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Error("获取用户主目录失败", logger.C(err))
			return nil, fmt.Errorf("获取用户主目录失败：%w", err)
		}
		memory.Cfg.MemPath = filepath.Join(home, "/mifer/memory", filepath.Base(config.Workdir))
	}
	// 加载已有的对话历史
	msgs, err := load(&memory.Cfg)
	if err != nil {
		logger.Error("加载记忆文件失败", logger.C(err))
		return nil, err
	}
	memory.Messages = msgs
	return &memory, nil
}
