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

var (
	pool          *redis.Pool
	redisServer   = flag.String("redisServer", ":6379", "")
	redisPassword = flag.String("redisPassword", os.Getenv("REDIS_PASSWORD"), "")
)

func main() {

	// Instantiate Vault Connection
	config := api.DefaultConfig()
	c, err := api.NewClient(config)
	if err != nil {
		log.Fatal("err: %s", err)
	}

	newToken, tokenErr := createVaultToken(c, "bjorn248@gmail.com")
	if tokenErr != nil {
		fmt.Println(tokenErr)
	}

	fmt.Println(newToken)

	flag.Parse()
	// Instantiate redis connection pool
	pool = newPool(*redisServer, *redisPassword)

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

func createVaultToken(c *api.Client, email string) (string, error) {
	createVaultPolicy(c, email)
	tcr := &api.TokenCreateRequest{
		Policies:    []string{email},
		DisplayName: email}
	ta := c.Auth().Token()
	s, err := ta.Create(tcr)
	if err != nil {
		return "", err
	}
	return s.Auth.ClientToken, nil
}

func createVaultPolicy(c *api.Client, email string) error {
	sys := c.Sys()
	rules := fmt.Sprintf("path \"secret/%s/*\" {\n  policy = \"write\"\n}", email)
	return sys.PutPolicy(email, rules)
}

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
