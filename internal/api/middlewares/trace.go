package middlewares

import (
	"mifer/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceMiddleware 为每个请求生成 traceID 并注入 context
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.New().String()
		ctx := c.Request.Context()
		ctx = logger.WithTraceID(ctx, traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("trace_id", traceID)

		// 将 traceID 写入响应头，方便客户端关联
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
