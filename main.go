package main

import (
	// 	"github.com/Bjorn248/ec2-stopwatch/interfaces"
	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.Default()

	router.GET("/user", getUser)
	router.POST("/register", register)

	// Listen on port 4000
	router.Run(":4000")
}
