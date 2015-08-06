package main

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
)

func verifyRegistrationToken(token string, st *StopwatchToken) (*StopwatchToken, error) {
	invalidToken := errors.New("Invalid Token")
	redisConn := pool.Get()
	defer redisConn.Close()
	verificationTokenHash := generateSha256String(token)
	verificationToken, redisError := redis.Values(redisConn.Do("HGETALL", verificationTokenHash))
	if redisError != nil {
		fmt.Sprintf("Error when looking up verification token: '%s'", redisError)
		return &StopwatchToken{}, redisError
	}

	if len(verificationToken) == 0 {
		return &StopwatchToken{}, invalidToken
	}
	_, redisError = redisConn.Do("DEL", verificationTokenHash)
	if redisError != nil {
		fmt.Sprintf("Error deleting redis data '%s'", redisError)
		return &StopwatchToken{}, redisError
	}
	if err := redis.ScanStruct(verificationToken, st); err != nil {
		return &StopwatchToken{}, err
	}
	if st.TokenType == "verification" {
		return st, nil
	} else {
		return &StopwatchToken{}, invalidToken
	}
}
