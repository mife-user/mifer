package agenthandler

import (
	"mifer/internal/api/dto/response/agentresp"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// LoadMemory 加载指定会话的记忆
func (h *AgentHandler) LoadMemory(c *gin.Context) {
	id := strings.TrimPrefix(c.Param("id"), "/")
	req := &domain.MemoryReq{ID: id}
	resp, err := h.getService().LoadMemory(c.Request.Context(), req)
	if err != nil {
		logger.Error("加载记忆失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.MemoryRes{Memory: resp.Memory})
}
