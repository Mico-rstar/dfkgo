package repository

import (
	"dfkgo/model"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(driver, source string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(source), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.User{}, &model.File{}, &model.Task{}); err != nil {
		return nil, err
	}
	return db, nil
}

func InitTestDB() (*gorm.DB, error) {
	dsn := os.Getenv("TEST_DB_SOURCE")
	if dsn == "" {
		dsn = "dfkgo:dfkgo123@tcp(127.0.0.1:3306)/dfkgo_test?charset=utf8mb4&parseTime=true&loc=Local"
	}
	return InitDB("mysql", dsn)
}
