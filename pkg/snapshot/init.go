package snapshot

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// New 创建快照服务。
// baseDir 为空时服务功能禁用，所有方法均为空操作。
func New(workdir, baseDir string) *Service {
	return &Service{
		workdir:      workdir,
		baseDir:      baseDir,
		lastManifest: make(Manifest),
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

// InitBaseline 创建初始快照 r0（仅在首次启动且 r0/manifest.json 不存在时执行）。
// 用于 reback 到第 1 轮时恢复至对话前的状态。
func (s *Service) InitBaseline() error {
	if s.baseDir == "" {
		return nil
	}

	r0Dir := filepath.Join(s.baseDir, "r0")
	manifestPath := filepath.Join(r0Dir, "manifest.json")

	// 新格式已存在，加载最近清单后直接返回（加载失败非致命，下次 SaveRound 将全量计算）
	if _, err := os.Stat(manifestPath); err == nil {
		_ = s.loadLatestManifest()
		return nil
	}

	// 检测旧格式全量快照并迁移
	if info, err := os.Stat(r0Dir); err == nil && info.IsDir() {
		if err := os.RemoveAll(r0Dir); err != nil {
			return err
		}
	}

	// 创建增量基线快照
	if err := s.SaveRound(0); err != nil {
		return err
	}

	return nil
}

// loadLatestManifest 扫描 baseDir 下所有轮次的清单，加载最大轮次到 lastManifest。
// 失败时返回错误但由调用方决定是否致命。
func (s *Service) loadLatestManifest() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return err
	}

	maxRound := -1
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "r") {
			continue
		}
		round, err := strconv.Atoi(entry.Name()[1:])
		if err != nil {
			continue
		}
		if round > maxRound {
			maxRound = round
		}
	}

	if maxRound < 0 {
		return nil
	}

	manifest, err := s.loadManifest(maxRound)
	if err != nil {
		return err
	}
	s.lastManifest = manifest
	return nil
}
