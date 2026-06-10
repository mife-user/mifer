package agenthandler

import (
	"net/http"

	agentres "mifer/internal/api/dto/response/agentresp"

	"github.com/gin-gonic/gin"
)

// ListAgents 处理 Agent 列表查询请求
func (h *AgentHandler) ListAgents(c *gin.Context) {
	resp, err := h.getService().ListAgents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agents := make([]agentres.AgentInfo, len(resp.Agents))
	for i, a := range resp.Agents {
		agents[i] = agentres.AgentInfo{
			Name:         a.Name,
			ModelBackend: a.ModelBackend,
			Provider:     a.Provider,
			Model:        a.Model,
			Description:  a.Description,
			Tools:        a.Tools,
		}
	}
	c.JSON(http.StatusOK, agentres.AgentListRes{Agents: agents})
}
