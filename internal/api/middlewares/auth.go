package middlewares

import (
	"mifer/pkg/auth"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		config := conf.GetConfig()
		// 从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("缺少认证头")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证token"})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && strings.ToLower(parts[0]) == "bearer") {
			logger.Warn("认证头格式错误")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证token
		claims, err := auth.ValidateToken(tokenString, config.JWT.Secret)
		if err != nil {
			logger.Warn("Token验证失败", logger.C(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token: " + err.Error()})
			c.Abort()
			return
		}
		if claims.UserID == 0 {
			logger.Warn("Token中缺少UserID")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token中缺少用户ID"})
			c.Abort()
			return
		}
		if claims.Name == "" {
			logger.Warn("Token中缺少用户名")
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
