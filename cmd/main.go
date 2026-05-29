package main

import (
	"fmt"
	"os"
	"time"

	"github.com/zzznow/z-3sp"
	"github.com/zzznow/z-3sp/internal"
	"github.com/gin-gonic/gin"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %v\n", r)
			time.Sleep(200 * time.Millisecond)
			os.Exit(1)
		}
	}()

	fmt.Println("z-3sp starting...")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "test"
	}
	fmt.Printf("env=%s config=config/application-%s.yml\n", env, env)

	if err := internal.InitConfig(env); err != nil {
		fmt.Fprintf(os.Stderr, "InitConfig failed: %v\n", err)
		time.Sleep(200 * time.Millisecond)
		os.Exit(1)
	}
	fmt.Println("config loaded")

	gin.SetMode(internal.Conf.Mode)
	r := gin.Default()
	handler.RegisterRoutes(r)

	addr := fmt.Sprintf("%s:%d", internal.Conf.Host, internal.Conf.Port)
	fmt.Printf("listening on %s\n", addr)
	if err := r.Run(addr); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		time.Sleep(200 * time.Millisecond)
		os.Exit(1)
	}
}
