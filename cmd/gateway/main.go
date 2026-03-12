package main

import (
	"banka-raf/internal/gateway"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	server, err := gateway.NewServer()
	if err != nil {
		log.Fatalf("Error connecting to services!")
	}
	gateway.SetupApi(router, server)
	router.Run(":8080")
}
