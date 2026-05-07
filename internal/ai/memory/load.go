package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

// load 加载记忆数据，文件不存在时自动创建并返回空列表
func load(cfg *MemCfg) ([]*schema.Message, error) {
	if err := os.MkdirAll(cfg.MemPath, 0755); err != nil {
		return nil, fmt.Errorf("创建内存目录失败：%w", err)
	}

	fileName := filepath.Join(cfg.MemPath, fmt.Sprintf("%s.json", cfg.id))
	data, err := os.ReadFile(fileName)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("读取文件失败：%w", err)
		}
		// 文件不存在，创建空记忆文件
		if err := os.WriteFile(fileName, []byte("[]"), 0644); err != nil {
			return nil, fmt.Errorf("创建记忆文件失败：%w", err)
		}
		return nil, nil
	}

	var messages []*schema.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("解析JSON失败：%w", err)
	}
	return messages, nil
}
