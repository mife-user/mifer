package agenthandler

import (
	"net/http"

	"mifer/internal/api/dto/response/agentresp"
	"mifer/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ListMemories 列出所有可用的记忆会话ID
func (h *AgentHandler) ListMemories(c *gin.Context) {
	resp, err := h.getService().ListMemories(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "列出记忆列表失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.MemoryListRes{
		Current: resp.Current,
		IDs:     resp.IDs,
	})
}
