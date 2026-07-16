package memory

import (
	"context"

	"mifer/pkg/logger"
	"os"
	"path/filepath"
	"strings"
)

// ListIDs 递归扫描记忆目录，返回所有可用的记忆 ID 列表（.jsonl 文件相对于 MemPath 的路径不含扩展名）。
// 兼容单层 ID（"12345"）和层级 ID（"qq_private/12345"）。
func (m *Memory) ListIDs() ([]string, error) {
	var ids []string
	err := filepath.WalkDir(m.Cfg.MemPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// 跳过快照目录（_snapshots 后缀），避免将 changes.jsonl 误识别为记忆文件
			if strings.HasSuffix(d.Name(), "_snapshots") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			rel, relErr := filepath.Rel(m.Cfg.MemPath, path)
			if relErr != nil {
				return relErr
			}
			// 排除 QQ 相关记忆（qq_ 前缀），仅列举普通会话
			if strings.HasPrefix(rel, "qq_") {
				return nil
			}
			ids = append(ids, strings.TrimSuffix(rel, ".jsonl"))
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		logger.Error(context.Background(), "读取记忆目录失败", logger.C(err))
		return nil, err
	}
	return ids, nil
}
