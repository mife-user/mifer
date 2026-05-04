package middlewares

import (
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 创建 CORS 中间件
func CORSMiddleware(config *conf.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, origin := range config.Gin.Cors.AllowOrigins {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		for _, method := range config.Gin.Cors.AllowMethods {
			c.Header("Access-Control-Allow-Methods", method)
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
