package main

import (
	"encoding/json"
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
	Region         string   `json:"region" binding:"required,eq=ap-northeast-1|eq=us-east-1|eq=us-west-1|eq=us-west-2|eq=eu-west-1|eq=eu-central-1|eq=ap-southeast-1|eq=ap-southeast-2|eq=sa-east-1"`
	InstanceID     string   `json:"instance_id" binding:"required"`
	AccessKeyID    string   `json:"access_key_id" binding:"required"`
	StartSchedule  schedule `json:"start" binding:"required"`
	EndSchedule    schedule `json:"end" binding:"required"`
	ExpirationDate int      `json:"expiration"`
}

type schedule struct {
	Minute     string `json:"minute" binding:"required"`
	Hour       string `json:"hour" binding:"required"`
	DayOfMonth string `json:"day_of_month" binding:"required"`
	Month      string `json:"month" binding:"required"`
	DayOfWeek  string `json:"day_of_week" binding:"required"`
}

// TODO Make this work with the user object stored in redis
// It does not currently
type User struct {
	aws map[string]map[string]map[string]*schedule
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

		_, redisError = redisConn.Do("SET", json.Email,
			`{ "verified": false }`)
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
		_, redisError = redisConn.Do("SET", verToken.Email,
			`{
				"verified": true,
				"aws": {},
				"joyent": {},
				"azure": {},
				"google": {}
			}`)
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
	var jsonRequestData ScheduleRequest

	if c.BindJSON(&jsonRequestData) == nil {

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

			path := fmt.Sprintf("secret/%s/aws/%s", SwToken.Email, jsonRequestData.AccessKeyID)

			secret, err := vclient.Logical().Read(path)
			if err != nil {
				fmt.Printf("error reading %s: %s", path, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error reading secrets from vault"})
			}

			awsSecret := secret.Data["secret_key"]
			fmt.Println(awsSecret)

			UserFromRedis, redisError := redis.String(redisConn.Do("GET", SwToken.Email))
			if redisError != nil {
				fmt.Printf("Error when looking up email: '%s'", redisError)
				return
			}

			var jsonData map[string]interface{}

			if err := json.Unmarshal([]byte(UserFromRedis), &jsonData); err != nil {
				fmt.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error parsing json from redis"})
				return
			}

			// TODO Get below code working with User struct instead of type-casting the interface{}
			//			user := &User{}
			//
			//			if err := json.Unmarshal([]byte(UserFromRedis), user); err != nil {
			//				fmt.Println(err)
			//				c.JSON(http.StatusInternalServerError, gin.H{
			//					"status": "Error reading secrets from vault"})
			//				return
			//			}
			//
			//			fmt.Printf("%+v\n", user)
			//			fmt.Printf("%#v\n", user)
			// fmt.Println(user.aws["PLACEHOLDER"]["PLACEHOLDER"].start)
			jsonData["aws"].(map[string]interface{})[jsonRequestData.AccessKeyID] = map[string]map[string]schedule{
				jsonRequestData.InstanceID: map[string]schedule{
					"start": jsonRequestData.StartSchedule,
					"stop":  jsonRequestData.EndSchedule,
				},
			}

			// JSON back to string after adding schedules
			jsonMarshaled, jsonError := json.Marshal(jsonData)
			if jsonError != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error parsing json"})
				return
			}

			// Back to map to add region
			if err := json.Unmarshal(jsonMarshaled, &jsonData); err != nil {
				fmt.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error parsing json"})
				return
			}

			// Add Region
			jsonData["aws"].(map[string]interface{})[jsonRequestData.AccessKeyID].(map[string]interface{})[jsonRequestData.InstanceID].(map[string]interface{})["region"] = jsonRequestData.Region

			// Back to string
			jsonMarshaled, jsonError = json.Marshal(jsonData)
			if jsonError != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error parsing json"})
				return
			}

			// TODO Add expiration, or create separate function to handle expiration

			fmt.Println(string(jsonMarshaled))

			// Write json back to redis
			_, redisError = redisConn.Do("SET", SwToken.Email, string(jsonMarshaled))
			if redisError != nil {
				fmt.Printf("Error inserting redis data '%s'", redisError)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": "Error inserting redis data"})
				return
			}

			// TODO Below is working POC code for starting an instance, need to move everything to cron
			/*
				response, err := stopInstance(jsonRequestData.AccessKeyID, awsSecret.(string), jsonRequestData.InstanceID, jsonRequestData.Region)
				if err != nil {
					fmt.Println(err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"status": "Error stopping ec2 instance"})
					return
				}

				fmt.Println(response)
			*/

			cronStringStart := fmt.Sprintf("0 %s %s %s %s %s", jsonRequestData.StartSchedule.Minute, jsonRequestData.StartSchedule.Hour, jsonRequestData.StartSchedule.DayOfMonth,
				jsonRequestData.StartSchedule.Month, jsonRequestData.StartSchedule.DayOfWeek)
			cronStringEnd := fmt.Sprintf("0 %s %s %s %s %s", jsonRequestData.EndSchedule.Minute, jsonRequestData.EndSchedule.Hour, jsonRequestData.EndSchedule.DayOfMonth,
				jsonRequestData.EndSchedule.Month, jsonRequestData.EndSchedule.DayOfWeek)

			cronScheduler.AddFunc(cronStringStart, func() {
				startInstance(jsonRequestData.AccessKeyID, awsSecret.(string), jsonRequestData.InstanceID, jsonRequestData.Region)
			})
			cronScheduler.AddFunc(cronStringEnd, func() {
				stopInstance(jsonRequestData.AccessKeyID, awsSecret.(string), jsonRequestData.InstanceID, jsonRequestData.Region)
			})

			fmt.Printf("%+v", cronScheduler.Entries())
			fmt.Printf("%#v", cronScheduler.Entries())

			c.JSON(http.StatusOK, gin.H{
				"status": "making progress"})

		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":     "problem verifying api token",
				"suggestion": "server load might be high right now, please try later"})
		}

	} else {
		fmt.Println(c.BindJSON(&jsonRequestData))
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "invalid request json"})
	}
}
