package logger

import (
	"context"

	"go.uber.org/zap"
)

// extractFields 从 context 中提取 traceID，追加到 fields 前面
func extractFields(ctx context.Context, fields []zap.Field) []zap.Field {
	tid := TraceID(ctx)
	if tid == "" {
		return fields
	}

	// 预分配容量，将 trace 置顶
	out := make([]zap.Field, 0, len(fields)+1)
	out = append(out, zap.String("trace_id", tid))
	out = append(out, fields...)
	return out
}

// Info 打印 info 日志
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Info(msg, extractFields(ctx, fields)...)
}

// Error 打印 error 日志
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Error(msg, extractFields(ctx, fields)...)
}

// Debug 打印 debug 日志
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Debug(msg, extractFields(ctx, fields)...)
}

// Warn 打印 warn 日志
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Warn(msg, extractFields(ctx, fields)...)
}

// Fatal 打印 fatal 日志
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Fatal(msg, extractFields(ctx, fields)...)
}
