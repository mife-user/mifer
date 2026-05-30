package executor

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// ListSkills 返回所有已加载的技能列表
func (e *Executor) ListSkills(ctx context.Context) (*domain.SkillListResp, error) {
	skillInfos := e.Humen.SkillManager.List()

	var skills []domain.SkillInfo
	for _, s := range skillInfos {
		skills = append(skills, domain.SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			Context:     s.Context,
			Agent:       s.Agent,
		})
	}

	if skills == nil {
		skills = []domain.SkillInfo{}
	}

	logger.Debug("技能列表查询完成")
	return &domain.SkillListResp{Skills: skills}, nil
}
