//go:build !production

package repository

import (
	"dfkgo/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	registerDialector("sqlite", func(source string) gorm.Dialector {
		return sqlite.Open(source)
	})
}

func InitTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&model.User{}, &model.File{}, &model.Task{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
