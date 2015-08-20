package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/uuid"
	"net/http"
)

type user struct {
	Email string `form:"email" json:"email" binding:"required"`
}

type awsSec struct {
	AccessKeyID string `form:"access_key_id" json:"access_key_id" binding:"required"`
	SecretKeyID string `form:"secret_key_id" json:"secret_key_id" binding:"required"`
}

// GET /private/user endpoint
// This function returns the user object of the user
// making the request
func getUser(c *gin.Context) {
	if SwTokenInterface, exists := c.Get("SwToken"); exists {
		SwToken := SwTokenInterface.(StopwatchToken)
		c.JSON(http.StatusOK, gin.H{
			"email": SwToken.Email})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":     "problem verifying api token",
			"suggestion": "server load might be high right now, please try later"})
	}
}

// POST /register endpoint
// the register function takes an email as the only input parameter and generates
// a UUID that it returns to the user
func register(c *gin.Context) {
	var json user

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

// POST /private/aws/secrets
func awsSecrets(c *gin.Context) {
	var json awsSec

	if c.BindJSON(&json) == nil {

		if SwTokenInterface, exists := c.Get("SwToken"); exists {
			SwToken := SwTokenInterface.(StopwatchToken)
			APIToken := GetStopwatchAPIToken(c)

			vconfig := api.DefaultConfig()
			vclient, verr := api.NewClient(vconfig)
			if verr != nil {
				fmt.Sprintf("Problem creating vault client, '%s'", verr)
			}

			vclient.SetToken(APIToken)

			// Check Vault Connection
			_, authError := vclient.Logical().Read("auth/token/lookup-self")
			if authError != nil {
				fmt.Sprintf("Something went wrong connecting to Vault! Error is '%s'", authError)
			}

			path := fmt.Sprintf("secret/%s/aws/%s", SwToken.Email, json.AccessKeyID)

			_, err := vclient.Logical().Write(path, map[string]interface{}{"secret_key": json.SecretKeyID})
			if err != nil {
				fmt.Sprintf("error writing %s: %s", path, err)
			} else {
				c.JSON(http.StatusOK, gin.H{
					"status": "secrets written"})
				fmt.Printf("wrote %s successfully", path)
			}

		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":     "problem verifying api token",
				"suggestion": "server load might be high right now, please try later"})
		}

	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "invalid request json"})
	}
}

// POST /private/aws/schedule
// This takes the start, end, and expiration (optional)
// times of an ec2 instance, lots of TODO here
func awsSchedule(c *gin.Context) {
}
