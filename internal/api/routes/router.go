package routes

import (
	"mifer/internal/api/middlewares"
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Router 路由结构体
type Router struct {
	config *conf.Config
	Redis  *redis.Client
}

// GetRouter 获取路由实例
func GetRouter() *Router {
	return &Router{}
}

// NewRouter 初始化路由
func (r *Router) NewRouter(config *conf.Config) bool {
	r.config = config
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
			ai.POST("/chat")
		}
	}

	return router
}
