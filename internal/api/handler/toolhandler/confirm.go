package toolhandler

import (
	"mifer/internal/ai/tools"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ConfirmHandler 工具确认 HTTP 处理器
type ConfirmHandler struct {
	bus *tools.ConfirmBus
}

// NewConfirmHandler 创建工具确认处理器
func NewConfirmHandler(bus *tools.ConfirmBus) *ConfirmHandler {
	return &ConfirmHandler{bus: bus}
}

// confirmReq 确认请求 DTO
type confirmReq struct {
	CallID string `json:"call_id" binding:"required"`
	Action string `json:"action" binding:"required"` // "accept" | "refuse" | "allow"
}

// Confirm 处理工具确认请求 POST /api/tool/confirm
func (h *ConfirmHandler) Confirm(c *gin.Context) {
	var req confirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := tools.ConfirmAction(req.Action)
	switch action {
	case tools.ActionAccept, tools.ActionRefuse, tools.ActionAllow:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 action: " + req.Action})
		return
	}

	ok := h.bus.Resolve(req.CallID, action)
	if !ok {
		c.JSON(http.StatusGone, gin.H{"error": "确认已过期或不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
