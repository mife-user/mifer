package agenthandler

import (
	"mifer/internal/api/dto/response/agentresp"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListPlans 列出所有可用的计划文件
func (h *AgentHandler) ListPlans(c *gin.Context) {
	resp, err := h.getService().ListPlans(c.Request.Context())
	if err != nil {
		logger.Error("列出计划文件失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.PlanListRes{Files: resp.Files})
}

// LoadPlan 加载指定计划文件的内容
func (h *AgentHandler) LoadPlan(c *gin.Context) {
	name := c.Param("name")
	resp, err := h.getService().LoadPlan(c.Request.Context(), name)
	if err != nil {
		logger.Error("加载计划文件失败", logger.C(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentres.PlanLoadRes{
		Name:    resp.Name,
		Content: resp.Content,
	})
}
