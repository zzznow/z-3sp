package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/sms")
	{
		v1.POST("/code", SendCode)
		v1.POST("/verify", VerifyCode)
	}
}
