package middlewares

import (
	"mifer/pkg/conf"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 创建 CORS 中间件
func CORSMiddleware(config *conf.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestOrigin := c.Request.Header.Get("Origin")
		allowOrigin := ""
		for _, o := range config.Gin.Cors.AllowOrigins {
			if o == "*" || o == requestOrigin {
				allowOrigin = requestOrigin
				if o == "*" && requestOrigin == "" {
					allowOrigin = "*"
				}
				break
			}
		}
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}
		c.Header("Access-Control-Allow-Methods", strings.Join(config.Gin.Cors.AllowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
