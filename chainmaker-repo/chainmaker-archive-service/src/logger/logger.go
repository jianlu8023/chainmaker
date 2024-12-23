/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package logger define log operation
package logger

import (
	"os"
	"strings"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// WrappedLogger log结构体
type WrappedLogger struct {
	*zap.SugaredLogger
}

// NewLogger init new logger
func NewLogger(moduleName string, logConfig *serverconf.LogConfig) *WrappedLogger {

	encoder := getEncoder()
	writeSyncer := getLogWriter(logConfig)

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

func getLogWriter(logConfig *serverconf.LogConfig) zapcore.WriteSyncer {

	hook := &lumberjack.Logger{
		Filename:   logConfig.LogPath,
		MaxSize:    logConfig.MaxSize,
		MaxBackups: logConfig.MaxBackups,
		MaxAge:     logConfig.MaxAge,
		Compress:   logConfig.Compress,
	}

	var syncer zapcore.WriteSyncer
	if logConfig.LogInConsole {
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
