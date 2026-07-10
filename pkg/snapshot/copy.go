package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveRound 保存第 round 轮的文件变更记录。
// 通过 size + mtime 快速判断未变更文件，仅对变更文件计算 SHA256 并写入 objects 池，
// 然后将变更条目追加到 changes.jsonl。若本轮无任何文件变更则不写入。
func (s *Service) SaveRound(round int) error {
	if s.baseDir == "" {
		return nil
	}

	currentFiles := make(map[string]bool) // 本轮遍历中存在的文件路径

	err := filepath.Walk(s.workdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(s.workdir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// 目录：跳过排除目录
		if info.IsDir() {
			if s.skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过快照目录自身
		if strings.Contains(relPath, "_snapshots") {
			return nil
		}

		currentFiles[relPath] = true

		// 快速变更检测：size + mtime 均未变 → 复用旧条目
		lastEntry, hasLast := s.lastManifest[relPath]
		currentMtime := info.ModTime().Unix()
		if hasLast && lastEntry.Size == info.Size() && lastEntry.Mtime == currentMtime {
			return nil
		}

		// 文件已变更 → 计算哈希并存入 objects 池
		hash, size, err := s.computeFileHash(path)
		if err != nil {
			return nil
		}

		if err := s.storeObject(path, hash); err != nil {
			return nil
		}

		entry := FileEntry{
			Path:  relPath,
			Hash:  hash,
			Size:  size,
			Mtime: currentMtime,
			Round: round,
		}

		if err := s.appendChange(entry); err != nil {
			return nil
		}

		s.lastManifest[relPath] = entry
		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历工作目录失败: %w", err)
	}

	// 检测已删除的文件：lastManifest 中存在但磁盘上已不存在
	for relPath := range s.lastManifest {
		if currentFiles[relPath] {
			continue
		}
		entry := FileEntry{
			Path:  relPath,
			Hash:  "", // 空哈希标记删除
			Round: round,
		}
		if err := s.appendChange(entry); err != nil {
			continue
		}
		delete(s.lastManifest, relPath)
	}

	return nil
}

// RestoreToRound 将工作目录恢复到指定轮次的状态。
// 扫描 changes.jsonl 中所有 round ≤ targetRound 的条目，对每个文件取最新条目，
// 从 objects 池恢复文件内容，并清理目标状态中不存在的多余文件。
func (s *Service) RestoreToRound(round int) error {
	if s.baseDir == "" {
		return nil
	}

	entries, err := s.readChanges()
	if err != nil {
		return fmt.Errorf("读取变更日志失败: %w", err)
	}

	// 构建目标状态：对每个 path，取 round ≤ targetRound 中 round 最大的条目
	targetState := make(map[string]FileEntry)
	for _, entry := range entries {
		if entry.Round > round {
			continue
		}
		existing, ok := targetState[entry.Path]
		if !ok || entry.Round > existing.Round {
			targetState[entry.Path] = entry
		}
	}

	// 第一步：将目标状态中的文件从 objects 池复制到工作目录
	for relPath, entry := range targetState {
		if entry.Hash == "" {
			continue // 已删除的文件不恢复
		}

		targetPath := filepath.Join(s.workdir, relPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			continue
		}

		srcPath := s.objectPath(entry.Hash)
		if err := s.copyFile(srcPath, targetPath); err != nil {
			continue
		}
	}

	// 第二步：清理工作目录中不在目标状态的多余文件
	filepath.Walk(s.workdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(s.workdir, path)
		if err != nil {
			return nil
		}
		if relPath == "." {
			return nil
		}

		// 跳过排除目录内的文件
		pathParts := strings.Split(relPath, string(os.PathSeparator))
		for _, part := range pathParts {
			if s.skipDirs[part] {
				return nil
			}
		}
		if strings.Contains(relPath, "_snapshots") {
			return nil
		}

		// 目标状态中不存在的文件 → 删除
		if _, ok := targetState[relPath]; !ok {
			_ = os.Remove(path)
		}
		return nil
	})

	return nil
}

// RemoveRound 删除第 round 轮的变更记录。
// 将 changes.jsonl 中该轮次的条目移除并原子重写（先写临时文件再 Rename）。
func (s *Service) RemoveRound(round int) {
	if s.baseDir == "" {
		return
	}

	entries, err := s.readChanges()
	if err != nil {
		return
	}

	// 过滤掉目标轮次的条目
	var kept []FileEntry
	for _, entry := range entries {
		if entry.Round != round {
			kept = append(kept, entry)
		}
	}

	// 无变化则跳过
	if len(kept) == len(entries) {
		return
	}

	changesPath := s.changesPath()
	tmpPath := changesPath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return
	}

	writeErr := func() error {
		defer f.Close()
		for _, entry := range kept {
			data, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			if _, err := f.Write(append(data, '\n')); err != nil {
				return err
			}
		}
		return nil
	}()

	if writeErr != nil {
		os.Remove(tmpPath)
		return
	}

	if err := os.Rename(tmpPath, changesPath); err != nil {
		os.Remove(tmpPath)
		return
	}

	// 重建内存中的最新清单
	_ = s.loadLatestManifest()
}
