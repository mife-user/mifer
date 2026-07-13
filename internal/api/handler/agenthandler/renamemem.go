package agenthandler

import (
	"mifer/internal/api/dto/request/agentreq"
	"mifer/internal/api/dto/response/agentresp"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RenameMemory 重命名当前会话记忆。
func (h *AgentHandler) RenameMemory(c *gin.Context) {
	var body agentreq.RenameMemoryReq
	if err := c.ShouldBindJSON(&body); err != nil {
		logger.Warn("解析重命名请求失败", logger.C(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
		return
	}
	if body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
		return
	}

	resp, err := h.getService().RenameMemory(c.Request.Context(), &domain.RenameMemoryReq{
		Name: body.Name,
	})
	if err != nil {
		logger.Error("重命名记忆失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.RenameMemoryRes{
		OldName: resp.OldName,
		NewName: resp.NewName,
		Message: "重命名成功: " + resp.OldName + " → " + resp.NewName,
	})
}
