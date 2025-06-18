package lib

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func GetRedisClient() *redis.Client {
	if redisClient != nil {
		return redisClient
	}
	redisHost := os.Getenv("REDIS_HOST")
	opt, err := redis.ParseURL(redisHost)
	if err != nil {
		log.Printf("[redis] Error parsing connection string: %s\n", err.Error())
		return nil
	}
	rdb := redis.NewClient(opt)
	redisClient = rdb
	return rdb
}

func TestRedis() {
	rdb := GetRedisClient()
	if err := rdb.Set(context.Background(), "test", "test", 0).Err(); err != nil {
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

// NewRedisClient Replace redis instance with custom client implementation
func NewRedisClient(c *redis.Client) *redis.Client {
	redisClient = c
	return redisClient
}
