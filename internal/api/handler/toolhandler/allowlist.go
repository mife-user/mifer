package toolhandler

import (
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/api/dto/response/agentresp"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AddAllowList 处理命令白名单添加 POST /api/tool/allowlist/add 请求。
// 将指定命令写入项目级 .mifer/allowlist.yaml，成功后该命令后续不再需要确认。
func (h *ToolHandler) AddAllowList(c *gin.Context) {
	var req agentreq.AddAllowListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn(c.Request.Context(), "解析白名单请求失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	if req.Command == "" {
		logger.Warn(c.Request.Context(), "命令不能为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少命令参数"})
		return
	}

	resp, err := h.getService().AddAllowList(c.Request.Context(), &domain.ToolAddAllowListReq{
		Command: req.Command,
	})
	if err != nil {
		logger.Error(c.Request.Context(), "添加命令到白名单失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入白名单失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, agentres.AddAllowListRes{
		Command: resp.Command,
		Added:   resp.Added,
	})
}
