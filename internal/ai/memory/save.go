package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Save 保存记忆数据到 JSON 文件
func (m *Memory) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 防止路径遍历
	if strings.Contains(m.Cfg.id, "..") || strings.Contains(m.Cfg.id, "/") || strings.Contains(m.Cfg.id, "\\") {
		return fmt.Errorf("id 包含非法字符: %s", m.Cfg.id)
	}
	// 创建文件夹
	if err := os.MkdirAll(m.Cfg.MemPath, 0755); err != nil {
		return fmt.Errorf("创建内存目录失败：%w", err)
	}

	// 写入 JSON 文件
	fileName := filepath.Join(m.Cfg.MemPath, fmt.Sprintf("%s.json", m.Cfg.id))
	data, err := json.Marshal(m.Messages)
	if err != nil {
		return fmt.Errorf("序列化JSON失败：%w", err)
	}
	if err := os.WriteFile(fileName, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败：%w", err)
	}
	return nil
}
