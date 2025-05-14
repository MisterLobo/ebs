package config

import (
	"fmt"
	"os"
)

// const dsn = "host=localhost user=postgres password=password dbname=ebsdb port=5432 sslmode=disable TimeZone=Asia/Manila"

func GetDSN() string {
	DATABASE_HOST := os.Getenv("DATABASE_HOST")
	DATABASE_PORT := os.Getenv("DATABASE_PORT")
	DATABASE_SSLMODE := os.Getenv("DATABASE_SSLMODE")
	DATABASE_TIMEZONE := os.Getenv("DATABASE_TIMEZONE")
	DATABASE_USER := os.Getenv("DATABASE_USER")
	DATABASE_PASSWORD := os.Getenv("DATABASE_PASSWORD")
	DATABASE_NAME := os.Getenv("DATABASE_NAME")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s", DATABASE_HOST, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME, DATABASE_PORT, DATABASE_SSLMODE, DATABASE_TIMEZONE)
	return dsn
}

const TIME_PARSE_FORMAT = "2006-01-02 15:04:05 -07:00"
