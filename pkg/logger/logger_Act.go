package logger

import "go.uber.org/zap"

// Info 打印info日志
func Info(msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Info(msg, fields...)
}

// Error 打印error日志
func Error(msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Error(msg, fields...)
}

// Debug 打印debug日志
func Debug(msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Debug(msg, fields...)
}

// Warn 打印warn日志
func Warn(msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Warn(msg, fields...)
}

// Fatal 打印fatal日志
func Fatal(msg string, fields ...zap.Field) {
	if loggerInstance == nil {
		return
	}
	loggerInstance.Fatal(msg, fields...)
}
