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

// Init 初始化日志实例，按级别分文件，支持按大小切割，终端彩色输出
func Init(config *conf.Config) error {
	var err error
	var logDir string

	if config.Env == "dev" {
		logDir = "./logs"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		logDir = filepath.Join(home, "/mifer/logs")
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	maxSizeMB := defaultInt(config.Log.MaxSize, 10)
	maxBackups := defaultInt(config.Log.MaxBackups, 10)
	env := config.Env

	// 文件编码器配置（无颜色）
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
	}

	// 控制台编码器配置（带颜色）
	consoleEncoderCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    "||\n",
		EncodeTime:    nowTime,
		EncodeLevel:   zapcore.CapitalColorLevelEncoder,
	}

	var fileEncoder, consoleEncoder zapcore.Encoder
	if env == "prod" {
		fileEncoder = zapcore.NewJSONEncoder(fileEncoderCfg)
	} else {
		fileEncoder = zapcore.NewConsoleEncoder(fileEncoderCfg)
	}
	consoleEncoder = zapcore.NewConsoleEncoder(consoleEncoderCfg)

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
	// 创建各级别日志核心，使用LevelOf精确匹配级别
	cores := []zapcore.Core{
		// Debug级别核心：仅记录Debug级别
		zapcore.NewCore(fileEncoder, debugFile, zapcore.LevelOf(zapcore.DebugLevel)),
		// Info级别核心：仅记录Info级别
		zapcore.NewCore(fileEncoder, infoFile, zapcore.LevelOf(zapcore.InfoLevel)),
		// Warn级别核心：仅记录Warn级别
		zapcore.NewCore(fileEncoder, warnFile, zapcore.LevelOf(zapcore.WarnLevel)),
		// Error及以上级别核心：记录Error、Panic、Fatal级别
		zapcore.NewCore(fileEncoder, errorFile, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})),
	}

	if env != "prod" {
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel)
		cores = append(cores, consoleCore)
	}

	tee := zapcore.NewTee(cores...)
	loggerInstance = zap.New(tee, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}
