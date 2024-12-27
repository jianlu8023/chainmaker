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
			Caller:      false,
		},
		glog.WithFileOutPut(),
		glog.WithConsoleOutPut(),
		glog.WithLumberjack(&glog.LumberjackConfig{
			FileName:   "./log/chainmaker-app.log",
			MaxSize:    2,
			MaxAge:     30,
			MaxBackups: 7,
			Compress:   false,
			Localtime:  true,
		}),
	)
)

func GetAppLogger() *zap.SugaredLogger {
	return appLogger
}
