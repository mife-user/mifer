package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"mifer/pkg/conf"
)

var loggerInstance *zap.Logger

func defaultInt(val, def int) int {
	if val <= 0 {
		return def
	}
	return val
}

// Init 初始化日志实例，按级别分文件，支持按大小切割，仅输出到文件
func Init() error {
	config := conf.GetConfig()
	var logDir string

	if config.Env == "dev" {
		logDir = "./logs"
	} else {
		logDir = filepath.Join(config.Path.CfgPath, "/logs")
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	maxSizeMB := defaultInt(config.Log.MaxSize, 10)
	maxBackups := defaultInt(config.Log.MaxBackups, 10)

	// 确定日志最低级别：配置显式指定优先，否则由 env 决定
	var minLevel zapcore.Level
	switch config.Log.Level {
	case "debug":
		minLevel = zapcore.DebugLevel
	case "info":
		minLevel = zapcore.InfoLevel
	case "warn":
		minLevel = zapcore.WarnLevel
	case "error":
		minLevel = zapcore.ErrorLevel
	default:
		if config.Env == "dev" {
			minLevel = zapcore.DebugLevel
		} else {
			minLevel = zapcore.InfoLevel
		}
	}

	// 文件编码器配置
	fileEncoderCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    "||\n",
		EncodeTime:    nowTime,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderCfg)

	// 打开各级别的切割文件
	debugFile, err := NewRotatingFile(filepath.Join(logDir, "debug.log"), maxSizeMB, maxBackups)
	if err != nil {
		return err
	}

	infoFile, err := NewRotatingFile(filepath.Join(logDir, "info.log"), maxSizeMB, maxBackups)
	if err != nil {
		debugFile.Close()
		return err
	}

	warnFile, err := NewRotatingFile(filepath.Join(logDir, "warn.log"), maxSizeMB, maxBackups)
	if err != nil {
		debugFile.Close()
		infoFile.Close()
		return err
	}
	errorFile, err := NewRotatingFile(filepath.Join(logDir, "error.log"), maxSizeMB, maxBackups)
	if err != nil {
		debugFile.Close()
		infoFile.Close()
		warnFile.Close()
		return err
	}

	// 创建各级别日志核心，使用LevelOf精确匹配级别，低于minLevel的级别不写入
	cores := []zapcore.Core{
		zapcore.NewCore(fileEncoder, debugFile, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == zapcore.DebugLevel && lvl >= minLevel
		})),
		zapcore.NewCore(fileEncoder, infoFile, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == zapcore.InfoLevel && lvl >= minLevel
		})),
		zapcore.NewCore(fileEncoder, warnFile, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == zapcore.WarnLevel && lvl >= minLevel
		})),
		zapcore.NewCore(fileEncoder, errorFile, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel && lvl >= minLevel
		})),
	}

	tee := zapcore.NewTee(cores...)
	loggerInstance = zap.New(tee, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}
