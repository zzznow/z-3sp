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
	fmt.Println("env:", env)
	if env == "" {
		env = "test"
	}
	fmt.Println("loading config for:", env)

	if err := internal.InitConfig(env); err != nil {
		fmt.Println("InitConfig failed:", err)
		panic(err)
	}
	fmt.Println("config loaded")

	if err := handler.InitSms(); err != nil {
		fmt.Println("InitSms failed:", err)
		panic(err)
	}
	fmt.Println("sms init done")
	if err := handler.InitRedis(); err != nil {
		fmt.Printf("warn: redis init failed: %v\n", err)
	}

	gin.SetMode(internal.Conf.Mode)
	r := gin.Default()
	handler.RegisterRoutes(r)

	addr := fmt.Sprintf("%s:%d", internal.Conf.Host, internal.Conf.Port)
	fmt.Printf("z-3sp SMS Service started at %s\n", addr)
	r.Run(addr)
}
