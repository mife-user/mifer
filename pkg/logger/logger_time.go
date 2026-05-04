package logger

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// nowTime 自定义时间编码器，将时间格式化为 "01-02 15:04:05.000"
func nowTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Local().Format("01-02 15:04:05.000"))
}
