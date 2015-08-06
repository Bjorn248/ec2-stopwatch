package main

import (
	"flag"
	"github.com/garyburd/redigo/redis"
	"os"
	"time"
)

// Shamelessly pasted from redigo example code
func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// Declare redis connection variables
var (
	pool          *redis.Pool
	redisServer   = flag.String("redisServer", ":6379", "")
	redisPassword = flag.String("redisPassword", os.Getenv("REDIS_PASSWORD"), "")
)
