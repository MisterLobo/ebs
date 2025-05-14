package db

import (
	"ebs/src/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func GetDb() *gorm.DB {
	if db != nil {
		return db
	}
	_db, err := gorm.Open(postgres.Open(config.GetDSN()))
	if err != nil {
		panic(err)
	}
	sqlDB, err := _db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	db = _db
	return _db
}
