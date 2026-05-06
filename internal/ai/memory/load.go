package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

// load 加载记忆数据
func load(cfg *MemCfg) ([]*schema.Message, error) {
	// 创建文件夹
	if err := os.MkdirAll(cfg.MemPath, 0755); err != nil {
		return nil, fmt.Errorf("创建内存目录失败：%w", err)
	}

	// 读取指定下的 JSON 文件
	var messages []*schema.Message
	fileName := filepath.Join(cfg.MemPath, fmt.Sprintf("%s.json", cfg.id))
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败：%w", err)
	}
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败：%w", err)
	}
	return messages, nil
}
