package main

import (
	"fmt"
	"os"

	"github.com/zzznow/z-3sp"
	"github.com/zzznow/z-3sp/internal"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("z-3sp starting...")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "test"
	}

	if err := internal.InitConfig(env); err != nil {
		fmt.Printf("InitConfig failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("config loaded")

	gin.SetMode(internal.Conf.Mode)
	r := gin.Default()
	handler.RegisterRoutes(r)

	addr := fmt.Sprintf("%s:%d", internal.Conf.Host, internal.Conf.Port)
	fmt.Printf("listening on %s\n", addr)
	if err := r.Run(addr); err != nil {
		fmt.Printf("server error: %v\n", err)
		os.Exit(1)
	}
}
