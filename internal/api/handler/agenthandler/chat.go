package agenthandler

import (
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *AgentHandler) Chat(c *gin.Context) {
	req := &agentreq.ChatReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.AgentService.Chat(c.Request.Context(), &domain.TalkReq{
		Content: req.Content,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Info("chat success", logger.S("resp", resp.Content))
	c.JSON(http.StatusOK, resp)
}
