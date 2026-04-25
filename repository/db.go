package repository

import (
	"dfkgo/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(driver, source string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dialector = mysql.Open(source)
	case "sqlite":
		dialector = sqlite.Open(source)
	default:
		dialector = mysql.Open(source)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// AutoMigrate 建表
	if err := db.AutoMigrate(&model.User{}, &model.File{}, &model.Task{}); err != nil {
		return nil, err
	}
	return db, nil
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
