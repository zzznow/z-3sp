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

	"github.com/zzznow/z-3sp/internal"
)

var smsClient *dysmsapi.Client
var rdb *redis.Client

func InitSms() error {
	cfg := internal.Conf.SmsConfig
	if cfg == nil || cfg.AccessKeyId == "" {
		slog.Warn("闃块噷浜戠煭淇￠厤缃负绌猴紝璺宠繃鍒濆鍖?)
		return nil
	}

	var err error
	smsClient, err = dysmsapi.NewClientWithAccessKey("cn-hangzhou", cfg.AccessKeyId, cfg.AccessKeySecret)
	if err != nil {
		return fmt.Errorf("鍒涘缓闃块噷浜戠煭淇″鎴风澶辫触: %w", err)
	}
	slog.Info("闃块噷浜戠煭淇″鎴风鍒濆鍖栨垚鍔?, "sign", cfg.SignName)
	return nil
}

func InitRedis() error {
	cfg := internal.Conf.RedisConfig
	if cfg == nil || cfg.Host == "" {
		slog.Warn("Redis 閰嶇疆涓虹┖锛岃烦杩囧垵濮嬪寲")
		return nil
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Passwd,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Redis 杩炴帴澶辫触", "error", err)
		rdb = nil
		return nil
	}
	slog.Info("Redis 鍒濆鍖栨垚鍔?)
	return nil
}

// 鈹€鈹€ DTOs 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

type SendSmsDTO struct {
	Phone string `json:"phone" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

type VerifySmsDTO struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

// 鈹€鈹€ Handlers 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func SendCode(c *gin.Context) {
	var req SendSmsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !isValidSmsType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "z-3sp: 涓嶆敮鎸佺殑鐭俊绫诲瀷"})
		return
	}

	// 棰戠巼闄愬埗
	if rdb == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "z-3sp: 鏈嶅姟涓嶅彲鐢?})
		return
	}
	intervalKey := "sms:interval:" + req.Type + ":" + req.Phone
	ok, _ := rdb.SetNX(c.Request.Context(), intervalKey, "1", 60*time.Second).Result()
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "z-3sp: 璇?0绉掑悗鍐嶈瘯"})
		return
	}

	// 鐢熸垚 6 浣嶉獙璇佺爜
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())

	// 鍙戦€佺煭淇?	if _, err := sendAliyunSms(req.Phone, code, req.Type); err != nil {
		slog.Error("鍙戦€佺煭淇″け璐?, "phone", req.Phone, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "z-3sp: 鍙戦€佸け璐ワ紝璇风◢鍚庨噸璇?})
		return
	}

	// 瀛樺偍
	codeKey := "sms:code:" + req.Type + ":" + req.Phone
	rdb.Set(c.Request.Context(), codeKey, code, 5*time.Minute)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "楠岃瘉鐮佸凡鍙戦€?}})
}

func VerifyCode(c *gin.Context) {
	var req VerifySmsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if rdb == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "z-3sp: 鏈嶅姟涓嶅彲鐢?})
		return
	}

	codeKey := "sms:code:" + req.Type + ":" + req.Phone
	stored, err := rdb.Get(c.Request.Context(), codeKey).Result()
	if err == redis.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "z-3sp: 楠岃瘉鐮佸凡杩囨湡"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "z-3sp: 绯荤粺閿欒"})
		return
	}
	if stored != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "z-3sp: 楠岃瘉鐮侀敊璇?})
		return
	}

	rdb.Del(c.Request.Context(), codeKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"verified": true, "phone": req.Phone}})
}

// 鈹€鈹€ Aliyun 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func sendAliyunSms(phone, code, smsType string) (string, error) {
	if smsClient == nil {
		return "", fmt.Errorf("鐭俊瀹㈡埛绔湭鍒濆鍖?)
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
