package logger

import (
	glog "github.com/jianlu8023/go-logger"
	"go.uber.org/zap"
)

var (
	appLogger = glog.NewSugaredLogger(
		&glog.Config{
			LogLevel:    "info",
			DevelopMode: true,
			StackLevel:  "error",
			ModuleName:  "[app]",
			Caller:      false,
		})
)

func GetAppLogger() *zap.SugaredLogger {
	return appLogger
}
