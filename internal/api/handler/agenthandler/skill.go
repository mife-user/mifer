package agenthandler

import (
	"net/http"

	agentres "mifer/internal/api/dto/response/agentresp"
	"mifer/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ListSkills 处理技能列表查询请求
func (h *AgentHandler) ListSkills(c *gin.Context) {
	resp, err := h.getService().ListSkills(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "获取技能列表失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	skills := make([]agentres.SkillInfo, len(resp.Skills))
	for i, s := range resp.Skills {
		skills[i] = agentres.SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			Context:     s.Context,
			Agent:       s.Agent,
		}
	}
	c.JSON(http.StatusOK, agentres.SkillListRes{Skills: skills})
}
