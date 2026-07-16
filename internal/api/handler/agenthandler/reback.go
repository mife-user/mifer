package agenthandler

import (
	"mifer/internal/api/dto/response/agentresp"
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
		logger.Error(c.Request.Context(), "获取回退列表失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	entries := make([]agentres.RebackEntryRes, len(resp.Entries))
	for i, e := range resp.Entries {
		entries[i] = agentres.RebackEntryRes{
			Index:   e.Index,
			Summary: e.Summary,
		}
	}
	c.JSON(http.StatusOK, agentres.RebackListRes{Entries: entries})
}

// Reback 回退到指定轮次的用户消息
func (h *AgentHandler) Reback(c *gin.Context) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index <= 0 {
		logger.Warn(c.Request.Context(), "解析回退索引失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的轮次索引: " + indexStr})
		return
	}
	req := &domain.RebackReq{Index: index}
	resp, err := h.getService().Reback(c.Request.Context(), req)
	if err != nil {
		logger.Error(c.Request.Context(), "回退对话失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.RebackRes{
		Summary: resp.Summary,
		Content: resp.Content,
		Message: resp.Message,
	})
}
