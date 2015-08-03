package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusOK, gin.H{"status": fmt.Sprintf("Hello Bjorn, your email is %s", json.Email)})
	}
}
