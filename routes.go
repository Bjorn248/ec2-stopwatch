package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/uuid"
	"net/http"
)

type RegistrationUser struct {
	Email string `json:"email" binding:"required"`
}

type awsSec struct {
	AccessKeyID string `json:"access_key_id" binding:"required"`
	SecretKeyID string `json:"secret_key_id" binding:"required"`
}

type ScheduleRequest struct {
	InstanceID     string   `json:"instance_id" binding:"required"`
	AccessKeyID    string   `json:"access_key_id" binding:"required"`
	StartSchedule  schedule `json:"start" binding:"required"`
	EndSchedule    schedule `json:"end"`
	ExpirationDate int      `json:"expiration"`
}

type schedule struct {
	Minute     string `json:"minute" binding:"required"`
	Hour       string `json:"hour" binding:"required"`
	DayOfMonth string `json:"day_of_month" binding:"required"`
	Month      string `json:"month" binding:"required"`
	DayOfWeek  string `json:"day_of_week" binding:"required"`
}

type User struct {
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
	var json RegistrationUser

	if c.BindJSON(&json) == nil {
		redisConn := pool.Get()
		defer redisConn.Close()

		redisReply, redisError := redis.Bool(redisConn.Do("EXISTS", json.Email))
		if redisError != nil {
			fmt.Printf("Error reading redis data '%s'", redisError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error reading redis data"})
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
			fmt.Printf("Error inserting redis data '%s'", redisError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error inserting redis data"})
			return
		}

		_, redisError = redisConn.Do("HMSET", json.Email,
			"verified", false)
		if redisError != nil {
			fmt.Printf("Error inserting redis data '%s'", redisError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error inserting redis data"})
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
			fmt.Printf("Error creating vault token '%s'", tokenErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error creating vault token"})
			return
		}
		apiTokenHash := generateSha256String(apiToken)

		_, redisError := redisConn.Do("HMSET", apiTokenHash,
			"email", verToken.Email,
			"tokenType", "api")
		if redisError != nil {
			fmt.Printf("Error inserting redis data '%s'", redisError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error inserting redis data"})
			return
		}
		_, redisError = redisConn.Do("HMSET", verToken.Email,
			"verified", true)
		if redisError != nil {
			fmt.Printf("Error inserting redis data '%s'", redisError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error inserting redis data"})
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
				fmt.Printf("Problem creating vault client, '%s'", verr)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error connecting to vault"})
				return
			}

			vclient.SetToken(APIToken)

			// Check Vault Connection
			_, authError := vclient.Logical().Read("auth/token/lookup-self")
			if authError != nil {
				fmt.Printf("Something went wrong connecting to Vault! Error is '%s'", authError)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Something went wrong connecting to vault"})
				return
			}

			path := fmt.Sprintf("secret/%s/aws/%s", SwToken.Email, json.AccessKeyID)

			_, err := vclient.Logical().Write(path, map[string]interface{}{"secret_key": json.SecretKeyID})
			if err != nil {
				fmt.Printf("error writing %s: %s", path, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error writing secrets to vault"})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"status": "secrets written"})
				fmt.Printf("wrote %s successfully\n", path)
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
// Made some progress here but this is incomplete
// More TODO
func awsSchedule(c *gin.Context) {
	var json ScheduleRequest

	if c.BindJSON(&json) == nil {

		if SwTokenInterface, exists := c.Get("SwToken"); exists {
			SwToken := SwTokenInterface.(StopwatchToken)
			APIToken := GetStopwatchAPIToken(c)
			redisConn := pool.Get()
			defer redisConn.Close()

			vconfig := api.DefaultConfig()
			vclient, verr := api.NewClient(vconfig)
			if verr != nil {
				fmt.Printf("Problem creating vault client, '%s'", verr)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error connecting to vault"})
				return
			}

			vclient.SetToken(APIToken)

			// Check Vault Connection
			_, authError := vclient.Logical().Read("auth/token/lookup-self")
			if authError != nil {
				fmt.Printf("Something went wrong connecting to Vault! Error is '%s'", authError)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Something went wrong connecting to vault"})
				return
			}

			path := fmt.Sprintf("secret/%s/aws/%s", SwToken.Email, json.AccessKeyID)

			secret, err := vclient.Logical().Read(path)
			if err != nil {
				fmt.Printf("error reading %s: %s", path, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error reading secrets from vault"})
			}

			awsSecret := secret.Data["secret_key"]
			fmt.Println(awsSecret)

			User, redisError := redis.StringMap(redisConn.Do("HGETALL", SwToken.Email))
			if redisError != nil {
				fmt.Printf("Error when looking up email: '%s'", redisError)
				return
			}

			fmt.Println(User)

			// User["aws"][json.AccessKeyID][json.InstanceID]["start"] = json.StartSchedule
			// if json.EndSchedule != nil {
			// 	User["aws"][json.AccessKeyID][json.InstanceID]["end"] = json.EndSchedule
			// }
			// if json.ExpirationDate != nil {
			// 	User["aws"][json.AccessKeyID][json.InstanceID]["expiration"] = json.ExpirationDate
			// }

			// fmt.Println(User)
			c.JSON(http.StatusOK, gin.H{
				"status": "making progress"})

		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":     "problem verifying api token",
				"suggestion": "server load might be high right now, please try later"})
		}

	} else {
		fmt.Println(c.BindJSON(&json))
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "invalid request json"})
	}
}
