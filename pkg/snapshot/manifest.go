package snapshot

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// changesPath 返回 changes.jsonl 的完整路径。
func (s *Service) changesPath() string {
	return filepath.Join(s.baseDir, "changes.jsonl")
}

// readChanges 读取 changes.jsonl 中的所有变更条目。
// 文件不存在时返回空切片（非错误）。
func (s *Service) readChanges() ([]FileEntry, error) {
	f, err := os.Open(s.changesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("打开变更日志失败: %w", err)
	}
	defer f.Close()

	var entries []FileEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry FileEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // 跳过损坏行
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取变更日志失败: %w", err)
	}
	return entries, nil
}

// appendChange 将单条变更记录追加写入 changes.jsonl。
// 使用 O_APPEND 保证追加操作的原子性（POSIX 保证 ≤ PIPE_BUF 的写入是原子的）。
func (s *Service) appendChange(entry FileEntry) error {
	f, err := os.OpenFile(s.changesPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开变更日志失败: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("序列化变更条目失败: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("写入变更日志失败: %w", err)
	}
	return nil
}

// objectPath 返回指定哈希对应的 objects 存储路径。
func (s *Service) objectPath(hash string) string {
	return filepath.Join(s.baseDir, "objects", hash[:2], hash)
}

// computeFileHash 计算文件 SHA256 并返回哈希值、文件大小。
func (s *Service) computeFileHash(filePath string) (hash string, size int64, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	written, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("读取文件失败 %s: %w", filePath, err)
	}

	return hex.EncodeToString(h.Sum(nil)), written, nil
}

// storeObject 将源文件以哈希为名存入 objects 池。若对象已存在则跳过。
func (s *Service) storeObject(srcPath, hash string) error {
	dstPath := s.objectPath(hash)

	// 对象已存在则跳过
	if _, err := os.Stat(dstPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	return s.copyFile(srcPath, dstPath)
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
