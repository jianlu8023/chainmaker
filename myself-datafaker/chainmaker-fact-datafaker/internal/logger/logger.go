package logger

import (
	glog "github.com/jianlu8023/go-logger"
	"go.uber.org/zap"
)

var (
	appLogger = glog.NewSugaredLogger(
		&glog.Config{
			LogLevel:    "debug",
			DevelopMode: true,
			StackLevel:  "error",
			ModuleName:  "[app]",
			Caller:      false,
		})

	sdkLogger = glog.NewSugaredLogger(
		&glog.Config{
			LogLevel:    "debug",
			ModuleName:  "[sdk]",
			Caller:      true,
			DevelopMode: true,
			StackLevel:  "error",
		},
		glog.WithConsoleFormat(),
		glog.WithFileOutPut(),
		glog.WithLumberjack(&glog.LumberjackConfig{
			FileName:   "./log/chainmaker-sdk.log",
			MaxSize:    2,
			MaxAge:     30,
			MaxBackups: 7,
			Compress:   true,
			Localtime:  true,
		}),
	)
)

func GetAppLogger() *zap.SugaredLogger {
	return appLogger
}

func GetSdkLogger() *zap.SugaredLogger {
	return sdkLogger
}
