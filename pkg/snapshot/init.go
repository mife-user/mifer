package snapshot

import (
	"os"
	"path/filepath"
)

// New 创建快照服务。
// baseDir 为空时服务功能禁用，所有方法均为空操作。
func New(workdir, baseDir string) *Service {
	return &Service{
		workdir:      workdir,
		baseDir:      baseDir,
		lastManifest: make(map[string]FileEntry),
		skipDirs: map[string]bool{
			".git":         true,
			"node_modules": true,
			".mifer":       true,
			"memory":       true, // 运行时记忆数据（避免快照覆盖自身）
			"config":       true, // 配置文件目录
			"logs":         true, // 日志文件目录
		},
	}
}

// InitBaseline 初始化快照基线。
// 若 changes.jsonl 已存在则加载最新状态到内存；否则检测并清理旧版 r{N}/ 目录，
// 通过 SaveRound(0) 创建全新的基线快照。
func (s *Service) InitBaseline() error {
	if s.baseDir == "" {
		return nil
	}

	// changes.jsonl 已存在，加载最新清单后直接返回（加载失败非致命，下次 SaveRound 将全量计算）
	if _, err := os.Stat(s.changesPath()); err == nil {
		_ = s.loadLatestManifest()
		return nil
	}

	// 检测并清理旧版 r{N}/ 目录（迁移）
	s.cleanLegacyRounds()

	// 创建基线快照
	if err := s.SaveRound(0); err != nil {
		return err
	}

	return nil
}

// cleanLegacyRounds 清理旧版 per-round 快照目录（r0/, r1/, ...）。
func (s *Service) cleanLegacyRounds() {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 旧版格式：目录名以 "r" 开头且后面全是数字
		if len(name) < 2 || name[0] != 'r' {
			continue
		}
		isRound := true
		for _, c := range name[1:] {
			if c < '0' || c > '9' {
				isRound = false
				break
			}
		}
		if isRound {
			_ = os.RemoveAll(filepath.Join(s.baseDir, name))
		}
	}
}

// loadLatestManifest 读取 changes.jsonl，重建 lastManifest 为每个文件的最新状态。
func (s *Service) loadLatestManifest() error {
	entries, err := s.readChanges()
	if err != nil {
		return err
	}

	s.lastManifest = make(map[string]FileEntry)
	for _, entry := range entries {
		if entry.Hash == "" {
			// 删除标记 → 从最新清单中移除
			delete(s.lastManifest, entry.Path)
		} else {
			s.lastManifest[entry.Path] = entry
		}
	}
	return nil
}
