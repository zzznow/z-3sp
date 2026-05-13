package main

import (
	"fmt"
	"os"

	"github.com/zzznow/z-3sp"
	"github.com/zzznow/z-3sp/internal"
	"github.com/gin-gonic/gin"
)

func main() {
	env := "test"
	if len(os.Args) > 1 {
		env = os.Args[1]
	}

	if err := internal.InitConfig(env); err != nil {
		panic(err)
	}

	if err := handler.InitSms(); err != nil {
		panic(err)
	}
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
