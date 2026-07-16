package toolhandler

import (
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/api/dto/response/agentresp"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Confirm 处理工具确认 POST /api/tool/confirm 请求。
func (h *ToolHandler) Confirm(c *gin.Context) {
	var req agentreq.ConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn(c.Request.Context(), "解析工具确认请求失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	if req.ID == "" {
		logger.Warn(c.Request.Context(), "确认请求ID为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少确认ID"})
		return
	}

	if req.Action != "confirm" && req.Action != "deny" && req.Action != "allow" {
		logger.Warn(c.Request.Context(), "无效的确认动作", logger.S("action", req.Action))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 action，应为 confirm/deny/allow"})
		return
	}

	resp, err := h.getService().Confirm(c.Request.Context(), &domain.ToolConfirmReq{
		ID:     req.ID,
		Action: req.Action,
	})
	if err != nil {
		logger.Error(c.Request.Context(), "处理工具确认失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Resolved {
		logger.Warn(c.Request.Context(), "确认条目未找到或已过期", logger.S("id", req.ID))
		c.JSON(http.StatusNotFound, gin.H{"error": "确认项未找到或已过期"})
		return
	}

	c.JSON(http.StatusOK, agentres.ConfirmRes{
		ID:       resp.ID,
		Resolved: resp.Resolved,
		Action:   resp.Action,
	})
}
