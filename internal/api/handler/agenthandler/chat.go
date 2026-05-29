package agenthandler

import (
	"context"
	"errors"
	"fmt"
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"
	"strings"
	"sync"
	"time"

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

	// 从请求 context 派生可取消的 context，心跳 goroutine 通过 cancel 退出
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 保护 c.Writer 的并发写入（心跳 goroutine 与业务回调 goroutine）
	var mu sync.Mutex

	// SSE 心跳：AI 推理/工具调用期间可能长时间无数据，
	// 每 15 秒发送 SSE comment（: heartbeat）防止中间网络设备因空闲断开连接
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				_, err := fmt.Fprintf(c.Writer, ": heartbeat\n\n")
				c.Writer.Flush()
				mu.Unlock()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	err := h.getService().Chat(ctx, &domain.TalkReq{
		Content: req.Content,
	}, func(event, content string) error {
		escaped := strings.ReplaceAll(content, "\n", "\\n")
		mu.Lock()
		_, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, escaped)
		c.Writer.Flush()
		mu.Unlock()
		if err != nil {
			// 写入失败说明客户端已断开，取消 context 通知主流程退出，
			// 返回 context.Canceled 让上层统一识别为"连接断开"
			cancel()
			return context.Canceled
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		logger.Error("chat失败", logger.C(err))
		mu.Lock()
		_, _ = fmt.Fprintf(c.Writer, "event: response\ndata: [ERROR] %s\n\n", err.Error())
		c.Writer.Flush()
		mu.Unlock()
		return
	}

	mu.Lock()
	_, _ = fmt.Fprintf(c.Writer, "event: response\ndata: [DONE]\n\n")
	c.Writer.Flush()
	mu.Unlock()
	logger.Info("chat success")
}
