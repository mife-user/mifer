package agenthandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListSkills 处理技能列表查询请求
func (h *AgentHandler) ListSkills(c *gin.Context) {
	resp, err := h.getService().ListSkills(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
