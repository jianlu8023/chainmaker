package mysql

import (
	"fmt"
	"time"

	"chainmaker-fact-datafaker/internal/database/entity"
	"chainmaker-fact-datafaker/internal/database/mysql/conn"
	"chainmaker-fact-datafaker/internal/logger"
	"github.com/jianlu8023/go-logger/dblogger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitConn() {
	// panic("当前使用配置有问题 请调整后注释panic")
	// mysqlDsn 数据库连接
	const (
		username = "root"
		password = "123456"
		host     = "172.25.138.54"
		port     = "3306"
		database = "chainmaker_archive_data"
	)

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%s?charset=utf8&parseTime=True&loc=Local",
		username,
		password,
		host,
		port,
		database)

	open, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: dblogger.NewDBLogger(dblogger.Config{
			LogLevel:                  dblogger.INFO,
			SlowThreshold:             200 * time.Microsecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			ShowSql:                   true,
		},
			dblogger.WithCustomLogger(logger.GetAppLogger().Desugar()),
		),
	})
	if err != nil {
		panic(err)
		return
	}
	conn.SetDb(open)
	migrate()
}

func migrate() {
	if err := conn.GetDb().AutoMigrate(&entity.FactEntity{}); err != nil {
		logger.GetAppLogger().Errorf("自动建表失败 %v", err)
	}
}
