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
	Action string `json:"action" binding:"required"` // "accept" | "refuse" | "allow" | "supplement"
	Input  string `json:"input,omitempty"`            // supplement 时的补充文本（仅 plan confirm 使用）
}

// Confirm 处理确认请求 POST /api/tool/confirm
//
// 同时处理工具确认（accept/refuse/allow）和计划确认（accept/refuse/supplement）。
// 通过 callID 前缀区分：plan_ 前缀走 ResolvePlan，否则走 Resolve。
func (h *ConfirmHandler) Confirm(c *gin.Context) {
	var req confirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := tools.ConfirmAction(req.Action)
	switch action {
	case tools.ActionAccept, tools.ActionRefuse, tools.ActionAllow, tools.ActionSupplement:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 action: " + req.Action})
		return
	}

	// 通过 callID 前缀区分计划确认和工具确认
	if isPlanCallID(req.CallID) {
		ok := h.bus.ResolvePlan(req.CallID, action, req.Input)
		if !ok {
			c.JSON(http.StatusGone, gin.H{"error": "确认已过期或不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	ok := h.bus.Resolve(req.CallID, action)
	if !ok {
		c.JSON(http.StatusGone, gin.H{"error": "确认已过期或不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// isPlanCallID 检查是否为计划确认的 callID
func isPlanCallID(id string) bool {
	return len(id) > 5 && id[:5] == "plan_"
}
