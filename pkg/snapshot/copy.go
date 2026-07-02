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
// 若本轮没有任何文件变更，则跳过 manifest 写入，避免产生大量内容相同的快照目录。
func (s *Service) SaveRound(round int) error {
	if s.baseDir == "" {
		return nil
	}

	newManifest := make(Manifest)
	var walkErrors []string
	changed := false

	err := filepath.Walk(s.workdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("访问路径失败 %s: %v", path, err))
			return nil
		}

		relPath, err := filepath.Rel(s.workdir, path)
		if err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("计算相对路径失败 %s: %v", path, err))
			return nil
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

		// 文件已变更，计算哈希并存入 objects 池
		changed = true
		hash, size, err := s.computeFileHash(path)
		if err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("计算文件哈希失败 %s: %v", path, err))
			return nil
		}

		if err := s.storeObject(path, hash); err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("存储快照对象失败 %s: %v", path, err))
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

	// 无文件变更则跳过写入，避免产生内容相同的快照目录
	if !changed {
		return nil
	}

	// 写出清单并更新内存缓存
	if err := s.writeManifest(round, newManifest); err != nil {
		return fmt.Errorf("写入快照清单失败: %w", err)
	}
	s.lastManifest = newManifest

	// 汇总遍历中的非致命错误
	if len(walkErrors) > 0 {
		return fmt.Errorf("快照保存部分失败: %s", strings.Join(walkErrors, "; "))
	}

	return nil
}

// RestoreToRound 恢复到 ≤ targetRound 的最近一次快照。
// 若 targetRound 对应的快照不存在（该轮未发生文件变更），自动向前查找最近的可用快照。
func (s *Service) RestoreToRound(targetRound int) error {
	if s.baseDir == "" {
		return nil
	}

	nearest := s.findNearestRound(targetRound)
	if nearest < 0 {
		return nil // 无任何快照，无需恢复
	}

	manifest, err := s.loadManifest(nearest)
	if err != nil {
		return fmt.Errorf("快照 r%d 不存在: %w", nearest, err)
	}

	var restoreErrors []string

	// 第一步：将清单中的文件从 objects 池复制到工作目录
	for relPath, entry := range manifest {
		targetPath := filepath.Join(s.workdir, relPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("创建目录失败 %s: %v", filepath.Dir(targetPath), err))
			continue
		}

		srcPath := s.objectPath(entry.Hash)
		if err := s.copyFile(srcPath, targetPath); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("恢复文件失败 %s: %v", relPath, err))
			continue
		}
	}

	// 第二步：清理工作目录中清单不存在的多余文件
	var cleanErrors []string
	_ = filepath.Walk(s.workdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			cleanErrors = append(cleanErrors, fmt.Sprintf("遍历清理路径失败 %s: %v", path, err))
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(s.workdir, path)
		if err != nil {
			cleanErrors = append(cleanErrors, fmt.Sprintf("计算相对路径失败 %s: %v", path, err))
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
			if err := os.Remove(path); err != nil {
				cleanErrors = append(cleanErrors, fmt.Sprintf("删除多余文件失败 %s: %v", relPath, err))
			}
		}
		return nil
	})

	// 汇总所有错误
	var allErrors []string
	allErrors = append(allErrors, restoreErrors...)
	allErrors = append(allErrors, cleanErrors...)
	if len(allErrors) > 0 {
		return fmt.Errorf("快照恢复部分失败: %s", strings.Join(allErrors, "; "))
	}

	// 更新内存缓存为当前快照
	s.lastManifest = manifest

	return nil
}

// RemoveRound 删除第 round 轮的快照，同时清理无引用的孤儿 objects。
// 若该轮快照不存在（SaveRound 跳过），则静默返回。
func (s *Service) RemoveRound(round int) error {
	if s.baseDir == "" {
		return nil
	}

	roundDir := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round))

	// 读取被删除轮次的清单，收集其独有的哈希
	removedManifest, err := s.loadManifest(round)
	if err != nil {
		// 清单不存在（该轮未发生文件变更），直接尝试删除目录后返回
		_ = os.RemoveAll(roundDir)
		return nil
	}

	// 收集所有剩余清单中的存活哈希（排除被删除轮次）
	liveHashes := s.collectLiveHashes(round)

	// 清理孤儿 objects：被删除轮次独有且无其他轮次引用的文件
	var removeErrors []string
	for _, entry := range removedManifest {
		if !liveHashes[entry.Hash] {
			if err := os.Remove(s.objectPath(entry.Hash)); err != nil && !os.IsNotExist(err) {
				removeErrors = append(removeErrors, fmt.Sprintf("删除孤儿对象失败 %s: %v", entry.Hash[:16], err))
			}
		}
	}

	// 删除清单目录
	if err := os.RemoveAll(roundDir); err != nil {
		removeErrors = append(removeErrors, fmt.Sprintf("删除快照目录失败 r%d: %v", round, err))
	}

	if len(removeErrors) > 0 {
		return fmt.Errorf("清理快照部分失败: %s", strings.Join(removeErrors, "; "))
	}

	return nil
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

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("关闭目标文件失败: %w", err)
	}

	return nil
}
