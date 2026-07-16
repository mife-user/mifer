package agenthandler

import (
	"net/http"

	agentres "mifer/internal/api/dto/response/agentresp"
	"mifer/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Compact 手动触发上下文压缩
func (h *AgentHandler) Compact(c *gin.Context) {
	resp, err := h.getService().Compact(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "手动压缩上下文失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.CompactRes{Message: resp.Message})
}
