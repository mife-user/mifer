package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

// load 从 JSONL 文件逐行加载记忆数据，文件不存在时返回空列表
func load(cfg *MemCfg) ([]*schema.Message, error) {
	if err := os.MkdirAll(cfg.MemPath, 0755); err != nil {
		return nil, fmt.Errorf("创建内存目录失败：%w", err)
	}

	fileName := filepath.Join(cfg.MemPath, fmt.Sprintf("%s.jsonl", cfg.id))
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("打开文件失败：%w", err)
	}
	defer f.Close()

	var messages []*schema.Message
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg schema.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			return nil, fmt.Errorf("解析行失败：%w", err)
		}
		messages = append(messages, &msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败：%w", err)
	}
	return messages, nil
}
