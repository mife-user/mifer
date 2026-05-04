package logger

import "go.uber.org/zap"

// Info 打印info日志
func Info(msg string, fields ...zap.Field) {
	loggerInstance.Info(msg, fields...)
}

// Error 打印error日志
func Error(msg string, fields ...zap.Field) {
	loggerInstance.Error(msg, fields...)
}

// Debug 打印debug日志
func Debug(msg string, fields ...zap.Field) {
	loggerInstance.Debug(msg, fields...)
}

// Warn 打印warn日志
func Warn(msg string, fields ...zap.Field) {
	loggerInstance.Warn(msg, fields...)
}

// Fatal 打印fatal日志
func Fatal(msg string, fields ...zap.Field) {
	loggerInstance.Fatal(msg, fields...)
}
