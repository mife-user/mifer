package agenthandler

import (
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListMemories 列出所有可用的记忆会话ID
func (h *AgentHandler) ListMemories(c *gin.Context) {
	resp, err := h.getService().ListMemories(c.Request.Context())
	if err != nil {
		logger.Error("列出记忆列表失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"current": resp.Current,
		"ids":     resp.IDs,
	})
}
