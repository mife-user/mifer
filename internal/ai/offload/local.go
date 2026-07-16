package offload

import (
	"context"
	"os"
	"path/filepath"

	"mifer/pkg/errorer"
	"mifer/pkg/logger"
)

// LocalOffloader 将工具结果卸载到本地文件系统。
// 文件存放于 baseDir/<key>，自动创建父目录。
type LocalOffloader struct {
	baseDir string
}

// NewLocal 创建本地文件系统 offloader，baseDir 为存储根目录。
func NewLocal(baseDir string) *LocalOffloader {
	return &LocalOffloader{baseDir: baseDir}
}

// Save 将 content 写入 baseDir/key 文件，返回绝对路径。
func (l *LocalOffloader) Save(ctx context.Context, key string, content []byte) (string, error) {
	fullPath := filepath.Join(l.baseDir, key)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		logger.Error(ctx, "创建 offload 目录失败", logger.S("path", filepath.Dir(fullPath)), logger.C(err))
		return "", errorer.NewS(errorer.ErrCreateMemoryDirFailed, err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		logger.Error(ctx, "写入 offload 文件失败", logger.S("path", fullPath), logger.C(err))
		return "", errorer.NewS(errorer.ErrWriteFileFailed, err)
	}

	logger.Debug(ctx, "Offload 文件已保存", logger.S("path", fullPath), logger.I("size", len(content)))
	return fullPath, nil
}

// Load 从 filePath 读取已保存的内容。
func (l *LocalOffloader) Load(ctx context.Context, filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error(ctx, "读取 offload 文件失败", logger.S("path", filePath), logger.C(err))
		return nil, errorer.NewS(errorer.ErrReadFileFailed, err)
	}
	return data, nil
}

// Delete 删除 filePath 对应的 offload 文件，文件不存在时静默成功。
func (l *LocalOffloader) Delete(ctx context.Context, filePath string) error {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		logger.Error(ctx, "删除 offload 文件失败", logger.S("path", filePath), logger.C(err))
		return errorer.NewS(errorer.ErrWriteFileFailed, err)
	}
	return nil
}

// BaseDir 返回 offloader 的存储根目录。
func (l *LocalOffloader) BaseDir() string {
	return l.baseDir
}
