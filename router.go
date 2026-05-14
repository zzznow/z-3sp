package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok")
	})
	v1 := r.Group("/sms")
	{
		v1.POST("/code", SendCode)
		v1.POST("/verify", VerifyCode)
	}
}
