package db

import (
	"ebs/src/config"
	"log"

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
		log.Printf("Error connecting to database: %s\n", err.Error())
		panic(err)
	}
	sqlDB, err := _db.DB()
	if err != nil {
		log.Fatalf("Error establishing connection to database: %s\n", err.Error())
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	db = _db
	return _db
}

func NewDB(newdb *gorm.DB) {
	db = newdb
}
