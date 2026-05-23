package agenthandler

import (
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/api/dto/response/agentresp"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetPrompt 获取当前系统提示词
func (h *AgentHandler) GetPrompt(c *gin.Context) {
	resp, err := h.AgentService.GetPrompt(c.Request.Context())
	if err != nil {
		logger.Error("获取提示词失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.PromptRes{Prompt: resp.Prompt})
}

// SetPrompt 设置自定义系统提示词
func (h *AgentHandler) SetPrompt(c *gin.Context) {
	var body agentreq.SetPromptReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
		return
	}
	if body.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提示词不能为空"})
		return
	}

	resp, err := h.AgentService.SetPrompt(c.Request.Context(), &domain.PromptReq{
		Prompt: body.Prompt,
	})
	if err != nil {
		logger.Error("设置提示词失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.PromptRes{Prompt: resp.Prompt})
}

// ResetPrompt 重置为默认系统提示词
func (h *AgentHandler) ResetPrompt(c *gin.Context) {
	resp, err := h.AgentService.ResetPrompt(c.Request.Context())
	if err != nil {
		logger.Error("重置提示词失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.PromptRes{Prompt: resp.Prompt})
}
