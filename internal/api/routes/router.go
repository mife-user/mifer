package routes

import (
	"context"
	"mifer/internal/ai/executor"
	"mifer/internal/api/handler/agenthandler"
	"mifer/internal/api/middlewares"
	"mifer/internal/service/agentservice"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"path/filepath"

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
func (r *Router) NewRouter(c context.Context, config *conf.Config) error {
	r.config = config

	exec, err := executor.Init(c, config)
	if err != nil {
		return err
	}
	service := agentservice.NewAgentService(exec)
	r.agentHandler = agenthandler.NewAgentHandler(service)

	return nil
}

// Setup 设置路由
func (r *Router) Setup() *gin.Engine {
	gin.SetMode(r.config.Gin.Mode)

	router := gin.Default()
	fileName := filepath.Join(r.config.Path.CfgPath, "/logs/gin.log")
	log, err := logger.NewRotatingFile(fileName, r.config.Log.MaxSize, r.config.Log.MaxBackups)
	if err == nil {
		gin.DefaultWriter = log
		gin.DefaultErrorWriter = log
	}

	router.Use(middlewares.CORSMiddleware(r.config))
	router.Use(middlewares.AuthMiddleware(r.config))
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
