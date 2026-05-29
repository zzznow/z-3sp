package main

import (
	"fmt"
	"os"

	"github.com/zzznow/z-3sp"
	"github.com/zzznow/z-3sp/internal"
	"github.com/gin-gonic/gin"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("FATAL: %v\n", r)
			os.Exit(1)
		}
	}()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "test"
	}

	if err := internal.InitConfig(env); err != nil {
		fmt.Printf("InitConfig failed: %v\n", err)
		os.Exit(1)
	}

	if err := handler.InitSms(); err != nil {
		fmt.Printf("warn: sms init failed: %v\n", err)
	}
	if err := handler.InitRedis(); err != nil {
		fmt.Printf("warn: redis init failed: %v\n", err)
	}

	gin.SetMode(internal.Conf.Mode)
	r := gin.Default()
	handler.RegisterRoutes(r)

	addr := fmt.Sprintf("%s:%d", internal.Conf.Host, internal.Conf.Port)
	fmt.Printf("z-3sp SMS Service started at %s\n", addr)
	if err := r.Run(addr); err != nil {
		fmt.Printf("server error: %v\n", err)
		os.Exit(1)
	}
}
