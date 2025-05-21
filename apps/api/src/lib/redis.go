package lib

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func GetRedisClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: "",
		DB:       0,
	})
	return rdb
}

func TestRedis() {
	rdb := GetRedisClient()
	err := rdb.Set(context.Background(), "test", "test", 0).Err()
	if err != nil {
		log.Printf("Failed to set value for key %s: %s\n", "test", err)
		return
	}
	val, err := rdb.Get(context.Background(), "test").Result()
	if err == redis.Nil {
		log.Println("No value")
		return
	} else if err != nil {
		log.Printf("Error retrieving value for test: %s\n", err.Error())
		return
	} else {
		log.Printf("Value is %v\n", val)
	}
}
