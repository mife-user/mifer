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
	agentHandler *agenthandler.AgentHandler
	appCtx       context.Context
}

// GetRouter 获取路由实例
func GetRouter() *Router {
	return &Router{}
}

// NewRouter 初始化路由
func (r *Router) NewRouter(c context.Context) error {
	exec, err := executor.Init(c)
	if err != nil {
		return err
	}
	service := agentservice.NewAgentService(exec)
	r.agentHandler = agenthandler.NewAgentHandler(service)
	r.appCtx = c

	return nil
}

// Setup 设置路由
func (r *Router) Setup() *gin.Engine {
	gin.SetMode(conf.GetConfig().Gin.Mode)

	fileName := filepath.Join(conf.GetConfig().Path.CfgPath, "/logs/gin.log")
	log, err := logger.NewRotatingFile(fileName, conf.GetConfig().Log.MaxSize, conf.GetConfig().Log.MaxBackups)
	if err == nil {
		gin.DefaultWriter = log
		gin.DefaultErrorWriter = log
	}

	router := gin.Default()
	router.Use(middlewares.CORSMiddleware())

	api := router.Group("/api")
	{
		ai := api.Group("/ai")
		{
			ai.POST("/chat", r.agentHandler.Chat)
		}
		memory := api.Group("/memory")
		{
			memory.GET("", r.agentHandler.ListMemories)
			memory.GET("/:id", r.agentHandler.LoadMemory)
			memory.POST("/exchange/:id", r.agentHandler.ExchangeMemory)
			memory.GET("/reback", r.agentHandler.ListRebackEntries)
			memory.POST("/reback/:index", r.agentHandler.Reback)
			memory.POST("/clear", r.agentHandler.ClearMemory)
		}
		prompt := api.Group("/prompt")
		{
			prompt.GET("", r.agentHandler.GetPrompt)
			prompt.POST("", r.agentHandler.SetPrompt)
			prompt.POST("/reset", r.agentHandler.ResetPrompt)
		}
		// Admin 管理接口
		admin := api.Group("/admin")
		{
			admin.POST("/reload", r.ReloadHandler)
		}
		plan := api.Group("/plan")
		{
			plan.GET("", r.agentHandler.ListPlans)
			plan.GET("/:name", r.agentHandler.LoadPlan)
		}
		mcp := api.Group("/mcp")
		{
			mcp.GET("/status", r.agentHandler.MCPStatus)
		}
		skill := api.Group("/skill")
		{
			skill.GET("/list", r.agentHandler.ListSkills)
		}
	}

	return router
}
