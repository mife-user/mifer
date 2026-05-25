package agenthandler

import (
	"mifer/internal/api/dto/response/agentresp"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClearMemory 生成新记忆会话ID并切换
func (h *AgentHandler) ClearMemory(c *gin.Context) {
	resp, err := h.getService().ClearMemory(c.Request.Context())
	if err != nil {
		logger.Error("清空记忆失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.ClearMemoryRes{NewID: resp.NewID})
}
