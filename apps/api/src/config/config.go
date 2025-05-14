package config

import (
	"os"
)

// const dsn = "host=localhost user=postgres password=password dbname=ebsdb port=5432 sslmode=disable TimeZone=Asia/Manila"

func GetDSN() string {
	DATABASE_CONNECTION := os.Getenv("DATABASE_CONNECTION")
	return DATABASE_CONNECTION
}

const TIME_PARSE_FORMAT = "2006-01-02 15:04:05 -07:00"
