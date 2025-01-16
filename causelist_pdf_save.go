package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

// Redis client
var ctx = context.Background()
var rdb *redis.Client

// Initialize Redis connection
func init() {
	// Connect to Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis address
		Password: "",               // No password set
		DB:       0,                // Default DB
	})

	// Test Redis connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
}

// saveSCCauseListToRedis saves the causeListMap data to Redis
func saveSCCauseListToRedis(causeListMap map[string]CauseList) {
	var causeListUniqueIDs []string

	// Iterate through the causeListMap and save each CauseList to Redis
	for key, causeList := range causeListMap {
		// Generate a unique ID based on the Description, DateOfHearing, and key
		causeListUniqueID := fmt.Sprintf("10-%s-%s-%s", causeList.Description, causeList.DateOfHearing, key)

		// Save to Redis using the unique ID as the key
		saveToRedis(causeListUniqueID, causeList)

		// Keep track of the saved unique IDs (optional)
		causeListUniqueIDs = append(causeListUniqueIDs, causeListUniqueID)

		fmt.Printf("Saved CauseList: %+v\n", causeList)
	}

	// Optionally, save mappings in Redis
	saveToRedisMapping(causeListUniqueIDs)
}

// saveToRedis saves each CauseList entry to Redis with a unique key
func saveToRedis(causeListUniqueID string, causeList CauseList) {
	// Use Redis SET to store the CauseList
	redisKey := fmt.Sprintf("causelist__%s", causeListUniqueID)
	err := rdb.Set(ctx, redisKey, fmt.Sprintf("%+v", causeList), 0).Err() // Storing CauseList as a string representation
	if err != nil {
		log.Printf("Error saving data to Redis: %v", err)
	}
}

// saveToRedisMapping saves the unique IDs to Redis (optional)
// saveToRedisMapping saves the unique IDs to Redis (optional)
// saveToRedisMapping saves the unique IDs to Redis (optional)
// saveToRedisMapping saves the unique IDs to Redis (optional)
func saveToRedisMapping(causeListUniqueIDs []string) {
	// Create slices to hold keys and values
	var keys []string
	var values []string

	// Populate keys and values
	for _, uniqueID := range causeListUniqueIDs {
		redisKey := fmt.Sprintf("causelist__%s", uniqueID)
		keys = append(keys, redisKey)
		values = append(values, uniqueID)
	}

	log.Println("causelistuniqueid", causeListUniqueIDs)
	// Create a slice to hold the flattened key-value pairs
	var pairs []interface{}
	for i := range keys {
		pairs = append(pairs, keys[i], values[i])
	}
	// Use MSET with the flattened slice
	err := rdb.MSet(ctx, pairs...).Err()
	if err != nil {
		log.Printf("Error saving multiple values to Redis: %v", err)
	}
}
