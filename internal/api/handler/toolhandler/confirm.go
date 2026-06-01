package toolhandler

import (
	"mifer/internal/ai/confirm"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ConfirmReq 工具确认请求体。
type ConfirmReq struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"` // "confirm" | "deny" | "allow"
}

// Confirm 处理工具确认 POST /api/tool/confirm 请求。
func (h *ToolHandler) Confirm(c *gin.Context) {
	var req ConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	if req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少确认ID"})
		return
	}

	var result confirm.ConfirmResult
	switch req.Action {
	case "confirm", "allow":
		result = confirm.ConfirmResult{Approved: true, Action: req.Action}
	case "deny":
		result = confirm.ConfirmResult{Approved: false, Action: "deny"}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 action，应为 confirm/deny/allow"})
		return
	}

	entry, ok := h.ConfirmStore.Get(req.ID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "确认项未找到或已过期"})
		return
	}

	h.ConfirmStore.Resolve(req.ID, result)

	logger.Info("工具确认已处理",
		logger.S("id", req.ID),
		logger.S("action", req.Action),
		logger.S("tool", entry.ToolName))

	c.JSON(http.StatusOK, gin.H{
		"id":       req.ID,
		"resolved": true,
		"action":   req.Action,
	})
}
