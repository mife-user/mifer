package agenthandler

import (
	"fmt"
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *AgentHandler) Chat(c *gin.Context) {
	req := &agentreq.ChatReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	err := h.AgentService.Chat(c.Request.Context(), &domain.TalkReq{
		Content: req.Content,
	}, func(content string) error {
		escaped := strings.ReplaceAll(content, "\n", "\\n")
		_, err := fmt.Fprintf(c.Writer, "data: %s\n\n", escaped)
		if err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		logger.Error("chat失败", logger.C(err))
		fmt.Fprintf(c.Writer, "data: [ERROR] %s\n\n", err.Error())
		c.Writer.Flush()
		return
	}

	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
	logger.Info("chat success")
}
