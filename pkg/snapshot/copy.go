package snapshot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mifer/pkg/logger"
)

// SaveRound 保存第 round 轮的工作目录快照到 {baseDir}/r{round}/。
func (s *Service) SaveRound(round int) error {
	if s.baseDir == "" {
		return nil
	}
	snapDir := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round))
	return copyDir(s.workdir, snapDir)
}

// RestoreToRound 从 {baseDir}/r{round}/ 恢复工作目录。
// 先复制快照文件覆盖工作目录，再清理工作目录中快照不存在的多余文件。
func (s *Service) RestoreToRound(round int) error {
	if s.baseDir == "" {
		return nil
	}
	snapDir := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round))
	// 检查快照目录是否存在
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return fmt.Errorf("快照 r%d 不存在: %s", round, snapDir)
	}
	return restoreDir(snapDir, s.workdir)
}

// RemoveRound 删除第 round 轮的快照目录。
func (s *Service) RemoveRound(round int) {
	if s.baseDir == "" {
		return
	}
	snapDir := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round))
	if err := os.RemoveAll(snapDir); err != nil {
		logger.Warn("删除快照目录失败", logger.C(err), logger.S("dir", snapDir))
	}
}

// copyDir 将 src 目录的全部文件复制到 dst 目录。
// 跳过 skipDirs 中指定的目录和 _snapshots 目录。
func copyDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		logger.Warn("删除旧快照目录失败", logger.C(err), logger.S("dst", dst))
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过快照目录自身
		if strings.Contains(relPath, "_snapshots") {
			return nil
		}

		targetPath := filepath.Join(dst, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return copyFile(path, targetPath)
	})
}

// restoreDir 从快照目录恢复工作目录。
// 先复制快照内容到工作目录（覆盖已有文件），再删除工作目录中快照不存在的多余文件。
func restoreDir(src, dst string) error {
	// 第一步：收集快照中的文件路径集合
	snapFiles := make(map[string]bool)
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return nil
		}
		snapFiles[relPath] = true
		return nil
	})

	// 第二步：将快照文件复制到工作目录（覆盖）
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		return copyFile(path, targetPath)
	}); err != nil {
		return err
	}

	// 第三步：清理工作目录中快照不存在的多余文件
	filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dst, path)
		if err != nil {
			return nil
		}
		if relPath == "." {
			return nil
		}
		// 跳过排除目录内的文件
		parts := strings.Split(relPath, string(os.PathSeparator))
		for _, part := range parts {
			if skipDirs[part] {
				return nil
			}
		}
		if strings.Contains(relPath, "_snapshots") {
			return nil
		}
		// 快照中不存在的文件 → 删除
		if !snapFiles[relPath] {
			if err := os.Remove(path); err != nil {
				logger.Warn("清理多余文件失败", logger.C(err), logger.S("file", relPath))
			}
		}
		return nil
	})

	return nil
}

// copyFile 复制单个文件内容。
func copyFile(src, dst string) error {
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
