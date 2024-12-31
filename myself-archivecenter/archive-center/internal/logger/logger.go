package logger

import (
	commonlog "chainmaker.org/chainmaker/common/v2/log"
	"go.uber.org/zap"
)

func GenChainMakerLogger(logInConsole bool) *zap.SugaredLogger {
	config := commonlog.LogConfig{
		Module:     "[SDK]",
		LogPath:    "./log/sdk.log",
		LogLevel:   commonlog.LEVEL_DEBUG,
		MaxAge:     30,
		JsonFormat: false,
		ShowLine:   true,
		// LogInConsole: false,
		// LogInConsole: true,
	}

	if logInConsole {
		config.LogInConsole = true
	} else {
		config.LogInConsole = false
	}

	log, _ := commonlog.InitSugarLogger(&config)
	return log
}
