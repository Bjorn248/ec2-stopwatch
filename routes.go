package main

import (
	// "encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/helper/uuid"
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

		apiToken, tokenErr := createVaultToken(vaultclient, json.Email)
		if tokenErr != nil {
			log.Print("Error creating vault token '%s'", tokenErr)
			return
		}

		// TODO Move this part of registration into verification
		// We don't need to save the apitoken in our database if the email
		// has not been verified yet
		// Also make sure to set this to valid: true during verification
		// Actually on second thought, move everything that doesn't need
		// to happen before verification into verification itself
		// - generating api token
		// - writing data in redis, determine which data to write?
		// - Store hash of apitoken instead of actual apitoken, during token verification, check hash
		_, redisError = redisConn.Do("HMSET", apiToken, "email", json.Email, "valid", false)
		if redisError != nil {
			log.Print("Error inserting redis data '%s'", redisError)
			return
		}

		verificationToken := uuid.GenerateUUID()

		// TODO Store hash of verification token, check hash during verification
		_, redisError = redisConn.Do("HMSET", verificationToken, "valid", true, "email", json.Email, "apiToken", apiToken)
		if redisError != nil {
			log.Print("Error inserting redis data '%s'", redisError)
			return
		}

		_, redisError = redisConn.Do("SET", json.Email, "true")
		if redisError != nil {
			log.Print("Error inserting redis data '%s'", redisError)
			return
		}

		go sendVerificationEmail(json.Email, verificationToken)

		c.JSON(http.StatusOK, gin.H{
			"status":       "user registered",
			"email":        json.Email,
			"email_status": "awaiting verification"})
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "invalid request json"})
	}
}

// GET /verify endpoint
// This function is used for email verification
// Where the clickable link provided to the user via email
// is handled
func verifyToken(c *gin.Context) {
	var st StopwatchToken
	token := c.Param("token")
	verToken, verTokenError := verifyRegistrationToken(token, &st)
	if verTokenError == nil {
		fmt.Sprintf("here we are %s", verToken.Email)
		go sendTokenEmail(verToken.Email, verToken.ApiToken)
		c.JSON(http.StatusOK, gin.H{
			"status":           "email verified",
			"api_token_status": fmt.Sprintf("email sent to %s", verToken.Email)})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "error verifying email",
			"error":  fmt.Sprintf("%s", verTokenError)})
	}
}
