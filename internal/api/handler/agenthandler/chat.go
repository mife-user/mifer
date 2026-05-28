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

	err := h.getService().Chat(c.Request.Context(), &domain.TalkReq{
		Content:  req.Content,
		PlanMode: req.PlanMode,
	}, func(event, content string) error {
		escaped := strings.ReplaceAll(content, "\n", "\\n")
		_, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, escaped)
		if err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		// 客户端断开或写入失败，不再尝试向已断开的连接写入
		logger.Warn("SSE写入失败，连接可能已断开", logger.C(err))
		return
	}

	fmt.Fprintf(c.Writer, "event: response\ndata: [DONE]\n\n")
	c.Writer.Flush()
	logger.Info("chat success")
}
