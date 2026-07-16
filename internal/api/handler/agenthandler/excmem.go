package agenthandler

import (
	"net/http"

	"mifer/internal/api/dto/response/agentresp"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/gin-gonic/gin"
)

func (h *AgentHandler) ExchangeMemory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		logger.Error(c.Request.Context(), errorer.ErrIdEmpty)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorer.ErrIdEmpty})
		return
	}
	req := &domain.MemoryReq{ID: id}
	if err := h.getService().ExchangeMemory(c.Request.Context(), req); err != nil {
		logger.Error(c.Request.Context(), "记忆交换失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.ExchangeMemoryRes{Message: "记忆交换成功"})
}
