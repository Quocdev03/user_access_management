// Package config cung cấp các cấu trúc dữ liệu và hàm tiện ích để nạp cấu hình hệ thống.
// Package này sử dụng Viper để đọc cấu hình từ file .env hoặc trực tiếp từ biến môi trường,
// giúp việc quản lý thiết lập cho ứng dụng (Database, Redis, JWT, Mail, Security) trở nên tập trung và nhất quán.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config chứa toàn bộ cấu hình tổng thể của ứng dụng, bao gồm các cấu hình thành phần.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Mail     MailConfig
	Security SecurityConfig
}

// AppConfig định nghĩa các thông số cơ bản để chạy ứng dụng (Môi trường, Cổng, URL frontend).
type AppConfig struct {
	Env         string
	Port        string
	FrontendURL string
}

// DatabaseConfig định nghĩa thông số kết nối tới cơ sở dữ liệu (MySQL).
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

// RedisConfig định nghĩa thông số kết nối tới Redis (dùng cho Session, Rate Limit).
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

// JWTConfig lưu trữ khóa bí mật và thời gian sống (TTL) của các token xác thực.
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// MailConfig định nghĩa thông số kết nối tới máy chủ SMTP để gửi email (OTP, Verify).
type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// SecurityConfig chứa các ngưỡng cấu hình liên quan đến bảo mật chống Spam và Brute-force.
type SecurityConfig struct {
	RateLimitRequests int
	RateLimitWindow   time.Duration
	MaxFailedAttempts int
	LockDuration      time.Duration
	OTPExpiry         time.Duration
	OTPMaxAttempts    int
}

// Load khởi tạo và nạp cấu hình từ file .env tại đường dẫn chỉ định hoặc lấy trực tiếp từ biến môi trường.
// Hàm này sử dụng cơ chế tự động ghi đè của Viper để ưu tiên biến môi trường (Environment Variables) hơn file .env.
// Trả về pointer của Config đã được nạp dữ liệu hoặc trả về lỗi nếu không thể phân tích cú pháp cấu hình.
func Load(path string) (*Config, error) {
	viper.SetConfigFile(path + "/.env")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
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
