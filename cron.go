package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/robfig/cron"
)

// Declare global cron scheduler
var cronScheduler *cron.Cron

func loadSchedulesFromRedis() error {
	redisConn := pool.Get()
	defer redisConn.Close()

	// Get all keys containing the @ symbol (i.e. emails)
	RedisKeys, redisError := redis.Strings(redisConn.Do("KEYS", "*@*"))
	if redisError != nil {
		fmt.Printf("Error when looking up email: '%s'", redisError)
		return redisError
	}

	// var jsonData map[string]interface{}

	fmt.Println(RedisKeys)

	// TODO Continue writing this function, finish it

	/*
		if err := json.Unmarshal([]byte(UserFromRedis), &jsonData); err != nil {
			fmt.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error parsing json from redis"})
			return
		}
	*/

	return nil
}
