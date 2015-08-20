package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	"log"
	"os"
)

func main() {

	// Check Environment Variables
	if os.Getenv("REDIS_PASSWORD") == "" {
		log.Fatal("REDIS_PASSWORD not set")
	}
	if os.Getenv("VAULT_TOKEN") == "" {
		log.Fatal("VAULT_TOKEN not set")
	}
	if os.Getenv("SENDGRID_API_TOKEN") == "" {
		log.Fatal("SENDGRID_API_TOKEN not set")
	}
	if os.Getenv("EMAIL_FROM_ADDRESS") == "" {
		log.Fatal("EMAIL_FROM_ADDRESS not set")
	}
	if os.Getenv("STOPWATCH_URL") == "" {
		log.Fatal("STOPWATCH_URL not set")
	}

	// Instantiate Vault Connection
	vaultconfig = api.DefaultConfig()
	vaultclient, vaulterror = api.NewClient(vaultconfig)
	if vaulterror != nil {
		log.Fatalf("Problem creating vault client, '%s'", vaulterror)
	}

	// Check Vault Connection
	_, authError := vaultclient.Logical().Read("auth/token/lookup-self")
	if authError != nil {
		log.Fatalf("Something went wrong connecting to Vault! Error is '%s'", authError)
	}

	flag.Parse()
	// Instantiate redis connection pool
	pool = newPool(*redisServer, *redisPassword)
	poolErr := pool.Get().Err()
	// Check redis connection
	if poolErr != nil {
		log.Fatalf("Something went wrong connecting to Redis! Error is '%s'", poolErr)
	}

	// Instantiate Gin Router
	router := gin.Default()

	private := router.Group("/private")

	private.Use(AuthRequired())
	{
		private.GET("/user", getUser)
		private.POST("/aws/secrets", awsSecrets)
		private.POST("/aws/schedule", awsSchedule)
	}

	router.POST("/register", register)
	router.GET("/verify/:token", verifyToken)

	// Listen on port 4000
	router.Run(":4000")
}
