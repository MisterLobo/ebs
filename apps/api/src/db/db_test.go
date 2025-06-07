package db

import (
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewMockDB() (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	testdb := "postgresql://postgres:password@localhost:5432/testdb?sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(testdb), &gorm.Config{
		ConnPool: db,
	})

	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening gorm database", err)
	}

	return gormDB, mock
}

func GetMockDB() (*gorm.DB, sqlmock.Sqlmock) {
	gormDB, mock := NewMockDB()
	db = gormDB
	return gormDB, mock
}

func TestDB(t *testing.T) {
	gormDB, _ := NewMockDB()
	db = gormDB

	assert.Equal(t, db.Name(), "testdb")
}
