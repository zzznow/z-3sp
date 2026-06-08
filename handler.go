package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/zzznow/common"
	"github.com/zzznow/z-3sp/internal"
)

var smsClient *dysmsapi.Client
var rdb *redis.Client

func InitSms() error {
	cfg := internal.Conf.SmsConfig
	if cfg == nil || cfg.AccessKeyId == "" {
		slog.Warn("_____________________________________________")
		return nil
	}

	var err error
	smsClient, err = dysmsapi.NewClientWithAccessKey("cn-hangzhou", cfg.AccessKeyId, cfg.AccessKeySecret)
	if err != nil {
		return fmt.Errorf("____________________________________: %w", err)
	}
	slog.Info("_______________________________________", "sign", cfg.SignName)
	return nil
}

func InitRedis() error {
	cfg := internal.Conf.RedisConfig
	if cfg == nil || cfg.Host == "" {
		slog.Warn("Redis ______________________________")
		return nil
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Passwd,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Redis ____________", "error", err)
		rdb = nil
		return nil
	}
	slog.Info("Redis _______________")
	return nil
}

//        DTOs                                                                                                                                                    

type SendSmsDTO struct {
	Phone string `json:"phone" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

type VerifySmsDTO struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

//        Handlers                                                                                                                                        

func SendCode(c *gin.Context) {
	var req SendSmsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorMsg(c, http.StatusBadRequest, err.Error())
		return
	}
	if !isValidSmsType(req.Type) {
		common.ErrorMsg(c, http.StatusBadRequest, "z-3sp: ________________________")
		return
	}

	//             
	if rdb == nil {
		common.ErrorMsg(c, http.StatusInternalServerError, "z-3sp: _______________")
		return
	}
	intervalKey := "sms:interval:" + req.Type + ":" + req.Phone
	ok, _ := rdb.SetNX(c.Request.Context(), intervalKey, "1", 60*time.Second).Result()
	if !ok {
		common.ErrorMsg(c, http.StatusTooManyRequests, "z-3sp: ___60____________")
		return
	}

	//        6             
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())

	//             
	if _, err := sendAliyunSms(req.Phone, code, req.Type); err != nil {
		slog.Error("__________________", "phone", req.Phone, "error", err)
		common.ErrorMsg(c, http.StatusInternalServerError, "z-3sp: ______________________________")
		return
	}

	//       
	codeKey := "sms:code:" + req.Type + ":" + req.Phone
	rdb.Set(c.Request.Context(), codeKey, code, 5*time.Minute)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "__________________"}})
}

func VerifyCode(c *gin.Context) {
	var req VerifySmsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorMsg(c, http.StatusBadRequest, err.Error())
		return
	}

	if rdb == nil {
		common.ErrorMsg(c, http.StatusInternalServerError, "z-3sp: _______________")
		return
	}

	codeKey := "sms:code:" + req.Type + ":" + req.Phone
	stored, err := rdb.Get(c.Request.Context(), codeKey).Result()
	if err == redis.Nil {
		common.ErrorMsg(c, http.StatusBadRequest, "z-3sp: __________________")
		return
	}
	if err != nil {
		common.ErrorMsg(c, http.StatusInternalServerError, "z-3sp: ____________")
		return
	}
	if stored != req.Code {
		common.ErrorMsg(c, http.StatusBadRequest, "z-3sp: _______________")
		return
	}

	rdb.Del(c.Request.Context(), codeKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"verified": true, "phone": req.Phone}})
}

//        Aliyun                                                                                                                                              

func sendAliyunSms(phone, code, smsType string) (string, error) {
	if smsClient == nil {
		return "", fmt.Errorf("___________________________")
	}

	cfg := internal.Conf.SmsConfig
	param, _ := json.Marshal(map[string]string{"code": code})

	req := dysmsapi.CreateSendSmsRequest()
	req.Scheme = "https"
	req.PhoneNumbers = phone
	req.SignName = cfg.SignName
	req.TemplateCode = cfg.TemplateCode
	req.TemplateParam = string(param)

	resp, err := smsClient.SendSms(req)
	if err != nil {
		return "", err
	}
	if resp.Code != "OK" {
		return "", fmt.Errorf("sms error: %s - %s", resp.Code, resp.Message)
	}
	return resp.BizId, nil
}

func isValidSmsType(t string) bool {
	switch t {
	case "login", "register", "reset_pwd", "bind_phone":
		return true
	}
	return false
}

