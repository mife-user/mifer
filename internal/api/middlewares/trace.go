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

		// 注入到 Go context（供 logger 等底层组件使用）
		ctx := c.Request.Context()
		ctx = logger.WithTraceID(ctx, traceID)

		// 尝试从请求头或查询参数中提取 sessionID
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			sessionID = c.Query("session_id")
		}
		if sessionID != "" {
			ctx = logger.WithSessionID(ctx, sessionID)
		}

		c.Request = c.Request.WithContext(ctx)

		// 同时存入 gin context，供 handler 层直接获取
		c.Set("trace_id", traceID)
		if sessionID != "" {
			c.Set("session_id", sessionID)
		}

		// 将 traceID 写入响应头，方便客户端关联
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
