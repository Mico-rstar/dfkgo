package repository

import (
	"dfkgo/model"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type dialectorFactory func(source string) gorm.Dialector

var extraDialectors = map[string]dialectorFactory{}

func registerDialector(name string, factory dialectorFactory) {
	extraDialectors[name] = factory
}

func InitDB(driver, source string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dialector = mysql.Open(source)
	default:
		if factory, ok := extraDialectors[driver]; ok {
			dialector = factory(source)
		} else {
			return nil, fmt.Errorf("unsupported db driver: %s", driver)
		}
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.User{}, &model.File{}, &model.Task{}); err != nil {
		return nil, err
	}
	return db, nil
}
