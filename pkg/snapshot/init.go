package snapshot

import (
	"os"
	"path/filepath"

	"mifer/pkg/logger"
)

// New 创建快照服务。
// baseDir 为空时服务功能禁用，所有方法均为空操作。
func New(workdir, baseDir string) *Service {
	return &Service{
		workdir: workdir,
		baseDir: baseDir,
	}
}

// InitBaseline 创建初始快照 r0（仅在首次启动且 r0 目录不存在时执行）。
// 用于 reback 到第 1 轮时恢复至对话前的状态。
func (s *Service) InitBaseline() error {
	if s.baseDir == "" {
		return nil
	}
	r0Dir := filepath.Join(s.baseDir, "r0")
	if _, err := os.Stat(r0Dir); os.IsNotExist(err) {
		logger.Info("创建初始文件快照 r0", logger.S("dir", r0Dir))
		if err := copyDir(s.workdir, r0Dir); err != nil {
			logger.Warn("创建初始快照 r0 失败，禁用快照功能", logger.C(err))
			s.baseDir = ""
			return err
		}
	}
	return nil
}
