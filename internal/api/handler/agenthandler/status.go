package agenthandler

import (
	"net/http"

	"mifer/internal/api/dto/response/adminresp"

	"github.com/gin-gonic/gin"
)

// Status 返回后端模型加载状态，供 CLI/TUI 启动时检查 AI 功能是否就绪
func (h *AgentHandler) Status(c *gin.Context) {
	resp, err := h.getService().BackendStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为 API 响应格式
	apiResp := &adminresp.StatusResp{
		Ready:    resp.Ready,
		Warnings: resp.Warnings,
		Backends: make([]adminresp.BackendStatus, 0, len(resp.Backends)),
	}
	for _, be := range resp.Backends {
		apiResp.Backends = append(apiResp.Backends, adminresp.BackendStatus{
			Name:   be.Name,
			Status: be.Status,
			Model:  be.Model,
			Error:  be.Error,
		})
	}

	c.JSON(http.StatusOK, apiResp)
}
