package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"net/http"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var st StopwatchToken
		stPointer := &st
		redisConn := pool.Get()
		defer redisConn.Close()
		token := GetStopwatchAPIToken(c)
		apiTokenHash := generateSha256String(token)
		apiToken, redisError := redis.Values(redisConn.Do("HGETALL", apiTokenHash))
		if redisError != nil {
			fmt.Printf("Error when looking up apiToken token: '%s'", redisError)
			return
		}
		if len(apiToken) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid api token"})
			c.Abort()
			return
		}

		if err := redis.ScanStruct(apiToken, stPointer); err != nil {
			return
		}

		if stPointer.TokenType == "api" {
			c.Set("SwToken", StopwatchToken{stPointer.Email, stPointer.TokenType})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid api token"})
			c.Abort()
			return
		}
	}
}

// Parses the api token header
func GetStopwatchAPIToken(c *gin.Context) string {
	if values, _ := c.Request.Header["Stopwatch-Api-Token"]; len(values) > 0 {
		return values[0]
	}
	return ""
}
