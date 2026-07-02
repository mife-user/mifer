package agenthandler

import (
	"context"
	"errors"
	"fmt"
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/sse"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *AgentHandler) Chat(c *gin.Context) {
	req := &agentreq.ChatReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		logger.Warn("解析聊天请求失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 限制输入长度，防止内存压力
	const maxContentLen = 200 * 1024 // 200KB
	if len(req.Content) > maxContentLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("消息内容过长（最大%dKB）", maxContentLen/1024)})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// sse.Writer 接管所有 SSE 写入 + 心跳
	fw, ok := c.Writer.(sse.FlushWriter)
	if !ok {
		logger.Error("ResponseWriter不支持Flush接口")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server streaming not supported"})
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
	sw := sse.New(ctx, fw, 10*time.Second, cancel)

	err := h.getService().Chat(ctx, &domain.TalkReq{
		Content: req.Content,
			Channel:  req.Channel,
			SessionID: req.SessionID,
	}, func(event, content string) error {
		escaped := strings.ReplaceAll(content, "\n", "\\n")
		return sw.SendSync(func(w sse.FlushWriter) error {
			_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, escaped)
			w.Flush()
			return err
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		logger.Error("chat失败", logger.C(err))
		if err := sw.SendSync(func(w sse.FlushWriter) error {
			_, err := fmt.Fprintf(w, "event: response\ndata: [ERROR] %s\n\n", err.Error())
			w.Flush()
			return err
		}); err != nil {
			logger.Warn("发送错误消息失败", logger.C(err))
		}
		return
	}

	if err := sw.SendSync(func(w sse.FlushWriter) error {
		_, err := fmt.Fprintf(w, "event: response\ndata: [DONE]\n\n")
		w.Flush()
		return err
	}); err != nil {
		logger.Warn("发送完成消息失败", logger.C(err))
	}
	logger.Info("chat success")
}
