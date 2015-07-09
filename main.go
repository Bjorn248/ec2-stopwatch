package main

import (
  "github.com/gin-gonic/gin"
  "net/http"
)

type registration struct {
  Email string `form:"email" json:"email" binding:"required"`
}

func main() {

  router := gin.Default()

  router.GET("/user", getUser)
  router.POST("/register", register)

  // Listen on port 4000
  router.Run(":4000")
}

func getUser(c *gin.Context) {
  c.String(http.StatusOK, "Hello Bjorn")
}

func register(c *gin.Context) {
  var json registration
  if c.BindJSON(&json) == nil {
    c.String(http.StatusOK, "Hello Bjorn, your email is %s", json.Email)
  }
}
