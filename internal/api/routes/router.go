package routes

import (
	"context"
	"mifer/internal/ai/executor"
	"mifer/internal/api/handler/agenthandler"
	"mifer/internal/api/middlewares"
	"mifer/internal/service/agentservice"
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
)

// Router 路由结构体
type Router struct {
	config       *conf.Config
	agentHandler *agenthandler.AgentHandler
}

// GetRouter 获取路由实例
func GetRouter() *Router {
	return &Router{}
}

// NewRouter 初始化路由
func (r *Router) NewRouter(c context.Context, config *conf.Config) bool {
	r.config = config
	executor, err := executor.Init(c, config)
	if err != nil {
		return false
	}
	service := agentservice.NewAgentService(executor)
	r.agentHandler = agenthandler.NewAgentHandler(service)

	return true
}

// Setup 设置路由
func (r *Router) Setup() *gin.Engine {
	gin.SetMode(r.config.Gin.Mode)

	router := gin.Default()

	router.Use(middlewares.CORSMiddleware(r.config))

	api := router.Group("/api")
	{
		ai := api.Group("/ai")
		{
			ai.POST("/chat", r.agentHandler.Chat)
		}
	}

	return router
}
