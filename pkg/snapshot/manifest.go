package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// loadManifest 读取指定轮次的快照清单。
func (s *Service) loadManifest(round int) (Manifest, error) {
	manifestPath := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round), "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("读取清单文件失败 %s: %w", manifestPath, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("解析清单文件失败 %s: %w", manifestPath, err)
	}
	return m, nil
}

// writeManifest 将快照清单写入指定轮次的 manifest.json（先写临时文件再 Rename 保证原子性）。
func (s *Service) writeManifest(round int, m Manifest) error {
	roundDir := filepath.Join(s.baseDir, fmt.Sprintf("r%d", round))
	if err := os.MkdirAll(roundDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化清单失败: %w", err)
	}

	manifestPath := filepath.Join(roundDir, "manifest.json")
	tmpPath := manifestPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("写入临时清单失败: %w", err)
	}

	if err := os.Rename(tmpPath, manifestPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名清单文件失败: %w", err)
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

// collectLiveHashes 扫描所有轮次的清单，收集仍在使用的哈希集合（排除指定轮次）。
func (s *Service) collectLiveHashes(excludeRound int) map[string]bool {
	liveHashes := make(map[string]bool)

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return liveHashes
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "r") {
			continue
		}

		// 解析轮次号
		roundStr := entry.Name()[1:] // 去掉 "r" 前缀
		round, err := strconv.Atoi(roundStr)
		if err != nil {
			continue
		}
		if round == excludeRound {
			continue
		}

		manifest, err := s.loadManifest(round)
		if err != nil {
			continue
		}

		for _, fileEntry := range manifest {
			liveHashes[fileEntry.Hash] = true
		}
	}

	return liveHashes
}

// findNearestRound 查找 ≤ maxRound 的最近一次快照轮次号。
// 若 r4 不存在但 r3 存在，返回 3；没有任何快照时返回 -1。
func (s *Service) findNearestRound(maxRound int) int {
	for r := maxRound; r > 0; r-- {
		manifestPath := filepath.Join(s.baseDir, fmt.Sprintf("r%d", r), "manifest.json")
		if _, err := os.Stat(manifestPath); err == nil {
			return r
		}
	}
	return -1
}
