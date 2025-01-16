package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConnection struct holds the Redis client
type RedisConnection struct {
	client *redis.Client
}

// NewRedisConnection initializes the Redis connection with environment variables
func NewRedisConnection() *RedisConnection {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisDBStr := os.Getenv("REDIS_DB")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Fatalf("Error parsing REDIS_DB: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		//Password: "",   // No password set (add if needed)
		DB:       redisDB, // Redis database number
	})

	// Test Redis connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}

	return &RedisConnection{client: rdb}
}

// SetValue sets a key-value pair in Redis with optional expiry time
func (r *RedisConnection) SetValue(key string, value interface{}, ex time.Duration) error {
	val, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshalling value: %v", err)
	}
	err = r.client.Set(ctx, key, val, ex).Err()
	if err != nil {
		return fmt.Errorf("error setting value in Redis: %v", err)
	}
	return nil
}

// GetValue retrieves a value from Redis by key
func (r *RedisConnection) GetValue(key string) (*CauseList, error) {
	value, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key does not exist")
	} else if err != nil {
		return nil, fmt.Errorf("error getting value from Redis: %v", err)
	}

	var causeList CauseList
	err = json.Unmarshal([]byte(value), &causeList)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling value: %v", err)
	}

	return &causeList, nil
}

// SetMultiValue sets multiple key-value pairs in Redis with optional expiry time
func (r *RedisConnection) SetMultiValue(data map[string]CauseList, ex time.Duration) error {
	pipe := r.client.Pipeline()
	for key, value := range data {
		val, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("error marshalling value: %v", err)
		}
		pipe.Set(ctx, key, val, ex)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error setting multi values in Redis: %v", err)
	}
	return nil
}

// GetMultiValue retrieves multiple values from Redis for the given keys
func (r *RedisConnection) GetMultiValue(keys []string) (map[string]CauseList, error) {
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting multi values from Redis: %v", err)
	}

	result := make(map[string]CauseList)
	for i, key := range keys {
		if values[i] != nil {
			var causeList CauseList
			err := json.Unmarshal([]byte(values[i].(string)), &causeList)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling value: %v", err)
			}
			result[key] = causeList
		}
	}
	return result, nil
}

