package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Mail     MailConfig
	Security SecurityConfig
}

type AppConfig struct {
	Env         string
	Port        string
	FrontendURL string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type SecurityConfig struct {
	RateLimitRequests int
	RateLimitWindow   time.Duration
	MaxFailedAttempts int
	LockDuration      time.Duration
	OTPExpiry         time.Duration
	OTPMaxAttempts    int
}

// Load đọc cấu hình từ file .env hoặc từ các biến môi trường
func Load(path string) (*Config, error) {
	viper.SetConfigFile(path + "/.env")

	viper.AutomaticEnv()
	// Thay thế dấu chấm bằng dấu gạch dưới cho biến môi trường nếu cần thiết
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		// Nếu không tìm thấy file config .env (do đang chạy trên server như Render),
		// hệ thống sẽ tự động sử dụng các biến môi trường của hệ điều hành.
		if !strings.Contains(err.Error(), "no such file or directory") {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	cfg := &Config{
		App: AppConfig{
			Env:         viper.GetString("APP_ENV"),
			Port:        viper.GetString("APP_PORT"),
			FrontendURL: viper.GetString("APP_FRONTEND_URL"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			Name:     viper.GetString("DB_NAME"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			AccessExpiry:  viper.GetDuration("JWT_ACCESS_EXPIRY"),
			RefreshExpiry: viper.GetDuration("JWT_REFRESH_EXPIRY"),
		},
		Mail: MailConfig{
			Host:     viper.GetString("SMTP_HOST"),
			Port:     viper.GetInt("SMTP_PORT"),
			User:     viper.GetString("SMTP_USER"),
			Password: viper.GetString("SMTP_PASSWORD"),
			From:     viper.GetString("SMTP_FROM"),
		},
		Security: SecurityConfig{
			RateLimitRequests: viper.GetInt("RATE_LIMIT_REQUESTS"),
			RateLimitWindow:   viper.GetDuration("RATE_LIMIT_WINDOW"),
			MaxFailedAttempts: viper.GetInt("MAX_FAILED_ATTEMPTS"),
			LockDuration:      viper.GetDuration("LOCK_DURATION"),
			OTPExpiry:         viper.GetDuration("OTP_EXPIRY"),
			OTPMaxAttempts:    viper.GetInt("OTP_MAX_ATTEMPTS"),
		},
	}

	return cfg, nil
}
