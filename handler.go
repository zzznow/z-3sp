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
		slog.Warn("阿里云短信配置为空，跳过初始化")
		return nil
	}

	var err error
	smsClient, err = dysmsapi.NewClientWithAccessKey("cn-hangzhou", cfg.AccessKeyId, cfg.AccessKeySecret)
	if err != nil {
		return fmt.Errorf("创建阿里云短信客户端失败: %w", err)
	}
	slog.Info("阿里云短信客户端初始化成功", "sign", cfg.SignName)
	return nil
}

func InitRedis() error {
	cfg := internal.Conf.RedisConfig
	if cfg == nil || cfg.Host == "" {
		slog.Warn("Redis 配置为空，跳过初始化")
		return nil
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Passwd,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Redis 连接失败", "error", err)
		rdb = nil
		return nil
	}
	slog.Info("Redis 初始化成功")
	return nil
}

// ── DTOs ─────────────────────────────────────────────────

type SendSmsDTO struct {
	Phone string `json:"phone" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

type VerifySmsDTO struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

// ── Handlers ─────────────────────────────────────────────

func SendCode(c *gin.Context) {
	var req SendSmsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !isValidSmsType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的短信类型"})
		return
	}

	// 频率限制
	if rdb == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务不可用"})
		return
	}
	intervalKey := "sms:interval:" + req.Type + ":" + req.Phone
	ok, _ := rdb.SetNX(c.Request.Context(), intervalKey, "1", 60*time.Second).Result()
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请60秒后再试"})
		return
	}

	// 生成 6 位验证码
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())

	// 发送短信
	if _, err := sendAliyunSms(req.Phone, code, req.Type); err != nil {
		slog.Error("发送短信失败", "phone", req.Phone, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送失败，请稍后重试"})
		return
	}

	// 存储
	codeKey := "sms:code:" + req.Type + ":" + req.Phone
	rdb.Set(c.Request.Context(), codeKey, code, 5*time.Minute)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "验证码已发送"}})
}

func VerifyCode(c *gin.Context) {
	var req VerifySmsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if rdb == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务不可用"})
		return
	}

	codeKey := "sms:code:" + req.Type + ":" + req.Phone
	stored, err := rdb.Get(c.Request.Context(), codeKey).Result()
	if err == redis.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统错误"})
		return
	}
	if stored != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	rdb.Del(c.Request.Context(), codeKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"verified": true, "phone": req.Phone}})
}

// ── Aliyun ───────────────────────────────────────────────

func sendAliyunSms(phone, code, smsType string) (string, error) {
	if smsClient == nil {
		return "", fmt.Errorf("短信客户端未初始化")
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
