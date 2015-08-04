package main

import (
	// "encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type registration struct {
	Email string `form:"email" json:"email" binding:"required"`
}

type User struct {
	Email string `form:"email" json:"email" binding:"required"`
}

// GET /user endpoint
// This function returns the user object of the user
// making the request
func getUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": "bjorn"})
}

// POST /register endpoint
// the register function takes an email as the only input parameter and generates a UUID that it returns to the user
func register(c *gin.Context) {
	var json registration

	if c.BindJSON(&json) == nil {
		redisConn := pool.Get()
		defer redisConn.Close()

		redisReply, redisError := redis.Bool(redisConn.Do("EXISTS", json.Email))
		if redisError != nil {
			log.Print("Error reading redis data '%s'", redisError)
		}
		if redisReply == true {
			c.JSON(http.StatusConflict, gin.H{
				"status": "user already exists",
				"email":  json.Email})
			return
		}

		newToken, tokenErr := createVaultToken(vaultclient, json.Email)
		if tokenErr != nil {
			log.Print("Error creating vault token '%s'", tokenErr)
		}

		_, redisError = redisConn.Do("HMSET", json.Email, "email", json.Email)
		if redisError != nil {
			log.Print("Error inserting redis data '%s'", redisError)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "user registered",
			"email":     json.Email,
			"api_token": newToken})
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "invalid request json"})
	}
}
