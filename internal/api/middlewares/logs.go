package middlewares

import (
	"github.com/gin-gonic/gin"
)

func LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Next()
	}

}
