package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/nazazulfiqi/be-mrt-schedules/modules/station"
)

func main() {
	InitiateRouter()
}

func InitiateRouter() {
	var (
		router = gin.Default()
		api    = router.Group("/v1/api")
	)

	station.Initiate(api)

	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0" // ip
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	router.Run(addr)
}
