package internal

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf = new(AppConfig)

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Mode        string `mapstructure:"mode"`
	Port        int    `mapstructure:"port"`
	Host        string `mapstructure:"host"`
	*RedisConfig `mapstructure:"redis"`
	*SmsConfig   `mapstructure:"sms"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	DB       int    `mapstructure:"db"`
	Passwd   string `mapstructure:"passwd"`
	PoolSize int    `mapstructure:"pool_size"`
}

type SmsConfig struct {
	AccessKeyId     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	SignName        string `mapstructure:"sign_name"`
	TemplateCode    string `mapstructure:"template_code"`
}

func InitConfig(env string) error {
	cfgFile := "config/application-" + env + ".yml"
	fmt.Printf("InitConfig: reading %s\n", cfgFile)

	viper.SetConfigFile(cfgFile)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	fmt.Println("InitConfig: viper read OK, setting env bindings...")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.BindEnv("redis.passwd")
	viper.BindEnv("sms.access_key_id")
	viper.BindEnv("sms.access_key_secret")

	fmt.Println("InitConfig: unmarshaling to struct...")
	if err := viper.Unmarshal(Conf); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	fmt.Printf("InitConfig: done, mode=%s port=%d host=%s\n", Conf.Mode, Conf.Port, Conf.Host)

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("InitConfig: config changed, reloading...")
		viper.Unmarshal(Conf)
	})
	return nil
}
