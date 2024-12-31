/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

package bfdb

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// GlobalLogger 日志记录
	GlobalLogger *WrappedLogger
)

// WrappedLogger 日志记录器
type WrappedLogger struct {
	*zap.SugaredLogger
}

// InitLogger 初始化日志
func InitLogger(cfg *LogConfig) {
	GlobalLogger = NewLogger("client-tools", cfg)
}

// NewLogger init new logger
func NewLogger(moduleName string, logConfig *LogConfig) *WrappedLogger {

	encoder := getEncoder()
	writeSyncer := getLogWriter(logConfig.LogPath, logConfig.LogInConsole)

	var level zapcore.Level
	switch strings.ToUpper(logConfig.LogLevel) {
	case "DEBUG":
		level = zapcore.DebugLevel
	case "INFO":
		level = zapcore.InfoLevel
	case "WARN":
		level = zapcore.WarnLevel
	case "ERROR":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		level,
	)

	logger := zap.New(core).Named(moduleName)
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	if logConfig.ShowColor {
		logger = logger.WithOptions(zap.AddCaller())
	}

	sugarLogger := logger.Sugar()

	return &WrappedLogger{
		sugarLogger,
	}
}

func getEncoder() zapcore.Encoder {

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    CustomLevelEncoder,
		EncodeTime:     CustomTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(logPath string, logInConsole bool) zapcore.WriteSyncer {

	hook := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}

	var syncer zapcore.WriteSyncer
	if logInConsole {
		syncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(hook))
	} else {
		syncer = zapcore.AddSync(hook)
	}

	return syncer
}

// CustomLevelEncoder 指定日志级别
func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

// CustomTimeEncoder 指定日志时间格式
func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// DebugDynamic debug
func (wl *WrappedLogger) DebugDynamic(getStr func() string) {
	wl.SugaredLogger.Debug(getStr())
}

// InfoDynamic info
func (wl *WrappedLogger) InfoDynamic(getStr func() string) {
	wl.SugaredLogger.Info(getStr())
}
