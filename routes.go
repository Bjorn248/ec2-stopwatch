package main

import (
	// "encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/helper/uuid"
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
			fmt.Sprintf("Error reading redis data '%s'", redisError)
		}
		if redisReply == true {
			c.JSON(http.StatusConflict, gin.H{
				"status": "user already exists",
				"email":  json.Email})
			return
		}

		verificationToken := uuid.GenerateUUID()
		verificationTokenHash := generateSha256String(verificationToken)

		_, redisError = redisConn.Do("HMSET", verificationTokenHash,
			"valid", true,
			"email", json.Email,
			"tokenType", "verification")
		if redisError != nil {
			fmt.Sprintf("Error inserting redis data '%s'", redisError)
			return
		}

		_, redisError = redisConn.Do("HMSET", json.Email,
			"verified", false)
		if redisError != nil {
			fmt.Sprintf("Error inserting redis data '%s'", redisError)
			return
		}

		go sendVerificationEmail(json.Email, verificationToken)

		c.JSON(http.StatusOK, gin.H{
			"status":                    "user registered",
			"email":                     json.Email,
			"email_verification_status": "pending verification",
			"verified":                  false})
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
		redisConn := pool.Get()
		defer redisConn.Close()
		apiToken, tokenErr := createVaultToken(vaultclient, verToken.Email)
		if tokenErr != nil {
			fmt.Sprintf("Error creating vault token '%s'", tokenErr)
			return
		}
		apiTokenHash := generateSha256String(apiToken)

		_, redisError := redisConn.Do("HMSET", apiTokenHash,
			"email", verToken.Email,
			"valid", true,
			"tokenType", "api")
		if redisError != nil {
			fmt.Sprintf("Error inserting redis data '%s'", redisError)
			return
		}
		_, redisError = redisConn.Do("HMSET", verToken.Email,
			"verified", true)
		if redisError != nil {
			fmt.Sprintf("Error inserting redis data '%s'", redisError)
			return
		}

		go sendTokenEmail(verToken.Email, apiToken)
		c.JSON(http.StatusOK, gin.H{
			"status":           "email verified",
			"verified":         true,
			"api_token_status": fmt.Sprintf("email sent to %s", verToken.Email)})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "error verifying email",
			"error":  fmt.Sprintf("%s", verTokenError)})
	}
}
