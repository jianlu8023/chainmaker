package mysql

import (
	"fmt"

	"chainmaker-fact-datafaker/internal/database/entity"
	"chainmaker-fact-datafaker/internal/database/mysql/conn"
	"chainmaker-fact-datafaker/internal/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitConn() {
	// mysqlDsn 数据库连接
	const (
		username = "chainmaker_archive_data"
		password = "chainmaker_archive_data_pw"
		host     = "192.168.58.110"
		port     = "3306"
		database = "chainmaker_archive_data"
	)

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, database)

	open, err := gorm.Open(mysql.Open(dsn))
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
