package snapshot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SaveRound 保存第 round 轮的工作目录快照。
// 通过 size + mtime 快速判断未变更文件（复用旧哈希），仅对变更文件计算 SHA256 并写入 objects 池。
func (s *Service) SaveRound(round int) error {
	if s.baseDir == "" {
		return nil
	}

	newManifest := make(Manifest)

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

		// 快速变更检测：size + mtime 均未变 → 复用旧哈希
		lastEntry, hasLast := s.lastManifest[relPath]
		currentMtime := info.ModTime().Unix()
		if hasLast && lastEntry.Size == info.Size() && lastEntry.Mtime == currentMtime {
			newManifest[relPath] = lastEntry
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

		newManifest[relPath] = FileEntry{
			Hash:  hash,
			Size:  size,
			Mtime: currentMtime,
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历工作目录失败: %w", err)
	}

	// 写出清单并更新内存缓存
	if err := s.writeManifest(round, newManifest); err != nil {
		return fmt.Errorf("写入快照清单失败: %w", err)
	}
	s.lastManifest = newManifest

	return nil
}

// RestoreToRound 从 manifest.json 恢复工作目录。
// 先将 objects 池中的文件复制到工作目录，再清理工作目录中清单不存在的多余文件。
func (s *Service) RestoreToRound(round int) error {
	if s.baseDir == "" {
		return nil
	}

	manifest, err := s.loadManifest(round)
	if err != nil {
		return fmt.Errorf("快照 r%d 不存在: %w", round, err)
	}

	// 第一步：将清单中的文件从 objects 池复制到工作目录
	for relPath, entry := range manifest {
		targetPath := filepath.Join(s.workdir, relPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			continue
		}

		srcPath := s.objectPath(entry.Hash)
		if err := s.copyFile(srcPath, targetPath); err != nil {
			continue
		}
	}

	// 第二步：清理工作目录中清单不存在的多余文件
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

		// 清单中不存在的文件 → 删除
		if _, ok := manifest[relPath]; !ok {
			_ = os.Remove(path)
		}
		return nil
	})

	return nil
}

// RemoveRound 删除第 round 轮的快照，同时清理无引用的孤儿 objects。
func (s *Service) RemoveRound(round int) {
	if s.baseDir == "" {
		return
	}

	roundDir := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round))

	// 读取被删除轮次的清单，收集其独有的哈希
	removedManifest, err := s.loadManifest(round)
	if err != nil {
		// 清单不存在或损坏，直接尝试删除目录
		_ = os.RemoveAll(roundDir)
		return
	}

	// 收集所有剩余清单中的存活哈希（排除被删除轮次）
	liveHashes := s.collectLiveHashes(round)

	// 清理孤儿 objects：被删除轮次独有且无其他轮次引用的文件
	for _, entry := range removedManifest {
		if !liveHashes[entry.Hash] {
			_ = os.Remove(s.objectPath(entry.Hash))
		}
	}

	// 删除清单目录
	_ = os.RemoveAll(roundDir)
}

// copyFile 复制单个文件内容。
func (s *Service) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}
