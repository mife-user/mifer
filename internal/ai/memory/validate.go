package memory

import (
	"mifer/pkg/errorer"
	"os"
	"path/filepath"
	"strings"
)

// validateID 验证会话 ID 合法性。
// 允许单层 ID（"12345"）和层级 ID（"qq_private/12345"），
// 拒绝路径遍历（".."）、绝对路径和空值。
func validateID(id string) error {
	if id == "" {
		return errorer.New(errorer.ErrIdEmpty)
	}
	if strings.Contains(id, "..") {
		return errorer.NewF(errorer.ErrIDIllegalChars, id)
	}
	cleaned := filepath.Clean(id)
	if cleaned == "." || cleaned == "" {
		return errorer.NewF(errorer.ErrIDIllegalChars, id)
	}
	if filepath.IsAbs(cleaned) {
		return errorer.NewF(errorer.ErrIDIllegalChars, id)
	}
	if strings.Contains(cleaned, "..") {
		return errorer.NewF(errorer.ErrIDIllegalChars, id)
	}
	return nil
}

// buildFilePath 根据会话 ID 构建 JSONL 文件完整路径，自动创建嵌套子目录。
func buildFilePath(memPath, id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}
	fullPath := filepath.Join(memPath, id+".jsonl")
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", errorer.NewS(errorer.ErrPathCannotCreate, err)
	}
	return fullPath, nil
}
