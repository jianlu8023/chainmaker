package conn

import (
	"gorm.io/gorm"
)

var db *gorm.DB

func SetDb(open *gorm.DB) {
	db = open
}

func GetDb() *gorm.DB {
	return db
}
