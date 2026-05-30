package executor

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"os"
	"path/filepath"
	"strings"
)

// ListPlans 列出 .mifer/plans/ 目录下的所有计划文件
func (e *Executor) ListPlans(ctx context.Context) (*domain.PlanListResp, error) {
	plansDir := filepath.Join(conf.GetConfig().Path.Workdir, ".mifer", "plans")

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("计划目录不存在，返回空列表", logger.C(err))
			return &domain.PlanListResp{Files: []string{}}, nil
		}
		logger.Error("读取计划目录失败", logger.C(err))
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	return &domain.PlanListResp{Files: files}, nil
}

// LoadPlan 加载指定计划文件的内容
func (e *Executor) LoadPlan(ctx context.Context, name string) (*domain.PlanLoadResp, error) {
	plansDir := filepath.Join(conf.GetConfig().Path.Workdir, ".mifer", "plans")
	filePath := filepath.Join(plansDir, name)

	// 安全检查：确保文件在 plansDir 内
	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}
	absDir, err := filepath.Abs(filepath.Clean(plansDir))
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(filepath.ToSlash(absPath), filepath.ToSlash(absDir)+"/") {
		logger.Warn("尝试读取计划目录外的文件: " + name)
		return nil, os.ErrNotExist
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("计划文件不存在: " + name)
		} else {
			logger.Error("读取计划文件失败", logger.C(err))
		}
		return nil, err
	}

	return &domain.PlanLoadResp{
		Name:    name,
		Content: string(content),
	}, nil
}
