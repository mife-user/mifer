package middlewares

import (
	"net/http"
	"strings"

	"mifer/pkg/auth"
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		config := conf.GetConfig()
		// 从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证token"})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && strings.ToLower(parts[0]) == "bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证token
			claims, err := auth.ValidateToken(tokenString, config.JWT.Secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token: " + err.Error()})
			c.Abort()
			return
		}
		if claims.UserID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token中缺少用户ID"})
			c.Abort()
			return
		}
		if claims.Name == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token中缺少用户名"})
			c.Abort()
			return
		}
		// 将用户信息存储到context中
		c.Set("user_id", claims.UserID)
		c.Set("user_name", claims.Name)

		c.Next()
	}
}
