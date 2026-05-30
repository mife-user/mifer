package agenthandler

import (
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
	c.JSON(http.StatusOK, resp)
}
