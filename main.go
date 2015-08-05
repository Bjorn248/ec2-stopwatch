package main

import (
	// "encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	"github.com/sendgrid/sendgrid-go"
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

type StopwatchToken struct {
	Valid    bool   `redis:"valid"`
	Email    string `redis:"email"`
	ApiToken string `redis:"apiToken"`
}

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

	router.GET("/user", getUser)
	router.POST("/register", register)
	router.GET("/verify/:token", verifyToken)

	// Listen on port 4000
	router.Run(":4000")
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

func verifyRegistrationToken(token string, st *StopwatchToken) (*StopwatchToken, error) {
	invalidToken := errors.New("Invalid Token")
	redisConn := pool.Get()
	defer redisConn.Close()
	verificationToken, redisError := redis.Values(redisConn.Do("HGETALL", token))
	if redisError != nil {
		fmt.Printf("Error when looking up verification token: '%s'", redisError)
		return &StopwatchToken{}, redisError
	}
	_, redisError = redisConn.Do("HMSET", token, "valid", "false", "apiToken", "")
	if redisError != nil {
		fmt.Printf("Error inserting redis data '%s'", redisError)
		return &StopwatchToken{}, redisError
	}
	if err := redis.ScanStruct(verificationToken, st); err != nil {
		return &StopwatchToken{}, err
	}
	if st.Valid == true {
		return st, nil
	} else {
		return &StopwatchToken{}, invalidToken
	}
}

func sendVerificationEmail(email, token string) {
	sg := sendgrid.NewSendGridClientWithApiKey(os.Getenv("SENDGRID_API_TOKEN"))
	message := sendgrid.NewMail()
	message.AddTo(email)
	message.SetSubject("Please Verify your Email for EC2 Stopwatch")
	// TODO Format this email a bit more
	// message.SetText("Please click the following link to verify your account.")
	message.SetHTML(fmt.Sprintf("<a href='%s/verify/%s'>%s/verify/%s</a>", os.Getenv("STOPWATCH_URL"), token, os.Getenv("STOPWATCH_URL"), token))
	message.SetFrom(os.Getenv("EMAIL_FROM_ADDRESS"))
	r := sg.Send(message)
	if r != nil {
		log.Print("Error sending email: '%s'", r)
		return
	}
}

func sendTokenEmail(email, token string) {
	sg := sendgrid.NewSendGridClientWithApiKey(os.Getenv("SENDGRID_API_TOKEN"))
	message := sendgrid.NewMail()
	message.AddTo(email)
	message.SetSubject("Your EC2 Stopwatch API Token")
	message.SetText(fmt.Sprintf("Your API Token is %s", token))
	message.SetFrom(os.Getenv("EMAIL_FROM_ADDRESS"))
	r := sg.Send(message)
	if r != nil {
		log.Print("Error sending email: '%s'", r)
		return
	}
}

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
