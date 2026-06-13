package toolhandler

import (
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

// AddAllowListReq 添加命令到白名单的请求体。
type AddAllowListReq struct {
	Command string `json:"command" binding:"required"`
}

// AddAllowList 处理命令白名单添加 POST /api/tool/allowlist/add 请求。
// 将指定命令写入项目级 .mifer/allowlist.yaml，成功后该命令后续不再需要确认。
func (h *ToolHandler) AddAllowList(c *gin.Context) {
	var req AddAllowListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("解析白名单请求失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	if req.Command == "" {
		logger.Warn("命令不能为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少命令参数"})
		return
	}

	// 检查是否已在白名单中
	existing, err := conf.LoadAllowList(h.Workdir)
	if err != nil {
		logger.Warn("加载命令白名单失败", logger.C(err))
	}
	if err == nil && slices.Contains(existing, req.Command) {
		c.JSON(http.StatusOK, gin.H{
			"command": req.Command,
			"added":   false,
			"message": "命令已在白名单中",
		})
		return
	}

	// 添加命令到白名单文件
	if err := conf.AddToAllowList(h.Workdir, req.Command); err != nil {
		logger.Error("添加命令到白名单失败",
			logger.S("command", req.Command),
			logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入白名单失败: " + err.Error()})
		return
	}

	logger.Info("命令已添加到白名单", logger.S("command", req.Command))
	c.JSON(http.StatusOK, gin.H{
		"command": req.Command,
		"added":   true,
	})
}
