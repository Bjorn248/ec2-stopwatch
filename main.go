package main

import (
	// "encoding/json"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	"log"
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

// Declare Vault Connection Variables
var (
	vaultconfig *api.Config
	vaultclient *api.Client
	vaulterror  error
)

func main() {

	// Check Environment Variables
	if os.Getenv("REDIS_PASSWORD") == "" {
		log.Fatal("REDIS_PASSWORD NOT SET")
	}
	if os.Getenv("VAULT_TOKEN") == "" {
		log.Fatal("VAULT_TOKEN NOT SET")
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

	testRedis()

	// Instantiate Gin Router
	router := gin.Default()

	router.GET("/user", getUser)
	router.POST("/register", register)

	// Listen on port 4000
	router.Run(":4000")
}

func testRedis() {
	conn := pool.Get()
	defer conn.Close()

	// set redis
	conn.Do("SET", "message1", "Hello Worldy")

	// get redis
	world, err := redis.String(conn.Do("GET", "message1"))
	if err != nil {
		fmt.Println("key not found")
		fmt.Println(err)
	}

	fmt.Println(world)
}

func createVaultToken(vaultclient *api.Client, email string) (string, error) {
	err := createVaultPolicy(vaultclient, email)
	if err != nil {
		log.Print("Error creating vault policy: '%s'", err)
	}
	tcr := &api.TokenCreateRequest{
		Policies:    []string{email},
		DisplayName: email}
	ta := vaultclient.Auth().Token()
	s, err := ta.Create(tcr)
	if err != nil {
		return "", err
	}
	return s.Auth.ClientToken, nil
}

func createVaultPolicy(vaultclient *api.Client, email string) error {
	sys := vaultclient.Sys()
	rules := fmt.Sprintf("path \"secret/%s/*\" {\n  policy = \"write\"\n}", email)
	return sys.PutPolicy(email, rules)
}

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
