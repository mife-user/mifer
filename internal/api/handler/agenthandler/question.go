package agenthandler

import (
	"net/http"

	"mifer/internal/domain"
	"mifer/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Answer 处理用户问答回复。
// POST /api/ai/answer
func (h *AgentHandler) Answer(c *gin.Context) {
	var req domain.AnswerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析问答回复请求失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "问题ID不能为空"})
		return
	}
	if req.Answer == "" && !req.IsSupplement {
		c.JSON(http.StatusBadRequest, gin.H{"error": "回答内容不能为空"})
		return
	}

	resp, err := h.getService().Answer(c.Request.Context(), &req)
	if err != nil {
		logger.Error("处理问答回复失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理失败"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
