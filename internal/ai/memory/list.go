package memory

import (
	"os"
	"strings"
)

// ListIDs 扫描记忆目录，返回所有可用的记忆ID列表（.jsonl文件名不含扩展名）
func (m *Memory) ListIDs() ([]string, error) {
	entries, err := os.ReadDir(m.Cfg.MemPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".jsonl") {
			ids = append(ids, strings.TrimSuffix(name, ".jsonl"))
		}
	}
	return ids, nil
}
