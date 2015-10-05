package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	// Check seal state of vault
	sealStatusResponse, sealStatusError := vaultclient.Sys().SealStatus()
	if sealStatusError != nil {
		log.Fatalf("Problem getting seal status from vault, '%s'", sealStatusError)
	}
	if sealStatusResponse.Sealed == true {
		// Provide 3 key shards to unseal the vault
		_, unsealError := vaultclient.Sys().Unseal(os.Getenv("VAULT_KEY_1"))
		if unsealError != nil {
			log.Fatalf("Problem unsealing vault, '%s'", unsealError)
		}
		_, unsealError = vaultclient.Sys().Unseal(os.Getenv("VAULT_KEY_2"))
		if unsealError != nil {
			log.Fatalf("Problem unsealing vault, '%s'", unsealError)
		}
		_, unsealError = vaultclient.Sys().Unseal(os.Getenv("VAULT_KEY_3"))
		if unsealError != nil {
			log.Fatalf("Problem unsealing vault, '%s'", unsealError)
		}
		sealStatusResponse, sealStatusError = vaultclient.Sys().SealStatus()
		if sealStatusError != nil {
			log.Fatalf("Problem getting seal status from vault, '%s'", sealStatusError)
		}
		if sealStatusResponse.Sealed == false {
			log.Println("Unsealed Vault")
		}
	}

	// Check Vault Connection
	_, authError := vaultclient.Logical().Read("auth/token/lookup-self")
	if authError != nil {
		log.Fatalf("Something went wrong connecting to Vault in main file! Error is '%s'", authError)
	}

	flag.Parse()
	// Instantiate redis connection pool
	pool = newPool(*redisServer, *redisPassword)
	poolErr := pool.Get().Err()
	// Check redis connection
	if poolErr != nil {
		log.Fatalf("Something went wrong connecting to Redis! Error is '%s'", poolErr)
	}

	// Instantiate Cron Scheduler
	cronScheduler = cron.New()
	cronScheduler.Start()

	// TODO Load all schedules from redis into scheduler on application start
	if redisErr := loadSchedulesFromRedis(); redisErr != nil {
		log.Fatalf("Something went wrong loading schedules from Redis: %s", redisErr)
	}

	// Instantiate Gin Router
	router := gin.Default()

	private := router.Group("/private")

	private.Use(AuthRequired())
	{
		private.GET("/user", getUser)
		private.POST("/aws/secrets", awsSecrets)
		private.POST("/aws/schedule", awsSchedule)
		// TODO Write DELETE Method for aws schedule
	}

	router.POST("/register", register)
	router.GET("/verify/:token", verifyToken)

	// Catch SIGINT and SIGTERM and Seal Vault
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for _ = range c {
			log.Print("Stopping Stopwatch...\nSealing Vault...\n")
			sealerror := vaultclient.Sys().Seal()
			if sealerror != nil {
				log.Fatalf("Error sealing vault: %s", sealerror)
			}
			log.Println("Stopwatch Shutdown Complete")
			os.Exit(0)
		}
	}()

	// Listen on port 4000
	router.Run(":4000")
}
