package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type registration struct {
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
		newToken, tokenErr := createVaultToken(vaultclient, json.Email)
		if tokenErr != nil {
			log.Fatal("err: %s", tokenErr)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "user registered",
			"email":     json.Email,
			"api_token": newToken})
	}
}
