package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// rotatingFile 实现 zapcore.WriteSyncer，支持按文件大小自动切割
type rotatingFile struct {
	mu          sync.Mutex
	file        *os.File
	path        string // 当前日志文件路径，如 ./logs/error.log
	maxSize     int64  // 单文件最大字节数
	maxBackups  int    // 保留备份文件最大数量
	currentSize int64  // 当前文件已写入字节数
}

// NewRotatingFile 创建切割文件写入器
func NewRotatingFile(path string, maxSizeMB int, maxBackups int) (*rotatingFile, error) {
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	if maxBackups <= 0 {
		maxBackups = 10
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &rotatingFile{
		file:        file,
		path:        path,
		maxSize:     int64(maxSizeMB) * 1024 * 1024,
		maxBackups:  maxBackups,
		currentSize: info.Size(),
	}, nil
}

func (r *rotatingFile) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentSize >= r.maxSize {
		if err := r.rotate(); err != nil {
			return 0, fmt.Errorf("rotate failed: %w", err)
		}
	}

	n, err = r.file.Write(p)
	r.currentSize += int64(n)
	return n, err
}

func (r *rotatingFile) rotate() error {
	if err := r.file.Close(); err != nil {
		return err
	}

	dir := filepath.Dir(r.path)
	base := filepath.Base(r.path)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var existingNums []int
	searchPrefix := prefix + "-"
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, searchPrefix) && strings.HasSuffix(name, ext) {
			numStr := strings.TrimSuffix(strings.TrimPrefix(name, searchPrefix), ext)
			if n, parseErr := strconv.Atoi(numStr); parseErr == nil {
				existingNums = append(existingNums, n)
			}
		}
	}
	sort.Ints(existingNums)

	nextNum := 1
	if len(existingNums) > 0 {
		nextNum = existingNums[len(existingNums)-1] + 1
	}

	backupPath := filepath.Join(dir, fmt.Sprintf("%s-%03d%s", prefix, nextNum, ext))
	if err := os.Rename(r.path, backupPath); err != nil {
		return err
	}

	newFile, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	r.file = newFile
	r.currentSize = 0

	allNums := append(existingNums, nextNum)
	sort.Ints(allNums)
	if excess := len(allNums) - r.maxBackups; excess > 0 {
		for _, num := range allNums[:excess] {
			delPath := filepath.Join(dir, fmt.Sprintf("%s-%03d%s", prefix, num, ext))
			os.Remove(delPath)
		}
	}

	return nil
}

func (r *rotatingFile) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Sync()
}

func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Close()
}
