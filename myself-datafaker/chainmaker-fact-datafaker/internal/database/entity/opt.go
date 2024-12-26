package entity

import (
	"errors"
	"fmt"

	"chainmaker-fact-datafaker/internal/database/mysql/conn"
	"gorm.io/gorm"
)

func (f FactEntity) Save() error {
	return save(f, conn.GetDb())
}

func save(f FactEntity, db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var exist FactEntity
		if err := tx.Model(&FactEntity{}).Where(&f).Find(&exist).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if exist.Id > 0 {
			return fmt.Errorf("record exist")
		}
		return tx.Create(&f).Error
	})
}
