package agenthandler

import (
	"mifer/internal/api/dto/response/agentresp"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MCPStatus 处理 MCP Server 状态查询请求
func (h *AgentHandler) MCPStatus(c *gin.Context) {
	resp, err := h.getService().MCPStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	servers := make([]agentres.MCPServerStatus, len(resp.Servers))
	for i, s := range resp.Servers {
		servers[i] = agentres.MCPServerStatus{
			Name:      s.Name,
			Status:    s.Status,
			ToolCount: s.ToolCount,
			Error:     s.Error,
		}
	}
	c.JSON(http.StatusOK, agentres.MCPStatusRes{Servers: servers})
}
