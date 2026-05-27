package agenthandler

import (
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListRebackEntries 返回当前会话中所有可回退的对话轮次
func (h *AgentHandler) ListRebackEntries(c *gin.Context) {
	resp, err := h.getService().ListRebackEntries(c.Request.Context())
	if err != nil {
		logger.Error("获取回退列表失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": resp.Entries})
}

// Reback 回退到指定轮次的用户消息
func (h *AgentHandler) Reback(c *gin.Context) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的轮次索引: " + indexStr})
		return
	}
	req := &domain.RebackReq{Index: index}
	resp, err := h.getService().Reback(c.Request.Context(), req)
	if err != nil {
		logger.Error("回退对话失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": resp.Summary, "content": resp.Content, "message": resp.Message})
}
