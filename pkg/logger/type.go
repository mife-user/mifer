package logger

import "context"

// 上下文键类型，避免跨包 key 冲突
type ctxKey struct{}

var traceKey = ctxKey{}

// WithTraceID 将 traceID 注入 context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey, traceID)
}

// TraceID 从 context 中提取 traceID，不存在时返回空字符串
func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(traceKey).(string)
	return v
}
