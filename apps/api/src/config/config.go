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

var (
	API_ENV             = os.Getenv("API_ENV")
	API_HOST            = os.Getenv("API_HOST")
	APP_HOST            = os.Getenv("APP_HOST")
	API_DOMAIN          = os.Getenv("API_DOMAIN")
	APP_DOMAIN          = os.Getenv("APP_DOMAIN")
	SMTP_FROM           = os.Getenv("SMTP_FROM")
	JWT_SECRET          = os.Getenv("JWT_SECRET")
	API_SECRET          = os.Getenv("API_SECRET")
	OAUTH_CLIENT_ID     = os.Getenv("OAUTH_CLIENT_ID")
	OAUTH_CLIENT_SECRET = os.Getenv("OAUTH_CLIENT_SECRET")
	GAPI_API_KEY        = os.Getenv("GAPI_API_KEY")
)
