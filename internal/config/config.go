// Package config cung cấp các cấu trúc dữ liệu và hàm tiện ích để nạp cấu hình hệ thống.
// Package này sử dụng Viper để đọc cấu hình từ file .env hoặc trực tiếp từ biến môi trường,
// giúp việc quản lý thiết lập cho ứng dụng (Database, Redis, JWT, Mail, Security) trở nên tập trung và nhất quán.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config chứa toàn bộ cấu hình tổng thể của ứng dụng, bao gồm các cấu hình thành phần.
type Config struct {
	App      AppConfig      `validate:"required"`
	Database DatabaseConfig `validate:"required"`
	Redis    RedisConfig    `validate:"required"`
	JWT      JWTConfig      `validate:"required"`
	Mail     MailConfig     `validate:"required"`
	Security SecurityConfig `validate:"required"`
}

// AppConfig định nghĩa các thông số cơ bản để chạy ứng dụng (Môi trường, Cổng, URL frontend).
type AppConfig struct {
	Env         string
	Port        string
	FrontendURL string
}

// DatabaseConfig định nghĩa thông số kết nối tới cơ sở dữ liệu (MySQL).
type DatabaseConfig struct {
	Host     string `validate:"required"`
	Port     string `validate:"required"`
	Name     string `validate:"required"`
	User     string `validate:"required"`
	Password string `validate:"required"`
}

// RedisConfig định nghĩa thông số kết nối tới Redis (dùng cho Session, Rate Limit).
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

// JWTConfig lưu trữ khóa bí mật và thời gian sống (TTL) của các token xác thực.
type JWTConfig struct {
	Secret        string        `validate:"required,min=16"`
	AccessExpiry  time.Duration `validate:"required"`
	RefreshExpiry time.Duration `validate:"required"`
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

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Kiểm tra các giá trị thời gian để tránh lỗi do viper parse plain number thành nanoseconds
	if cfg.JWT.AccessExpiry < time.Second ||
		cfg.JWT.RefreshExpiry < time.Second ||
		cfg.Security.RateLimitWindow < time.Second ||
		cfg.Security.LockDuration < time.Second ||
		cfg.Security.OTPExpiry < time.Second {
		return nil, fmt.Errorf("cấu hình thời gian (JWT_ACCESS_EXPIRY, RATE_LIMIT_WINDOW, v.v.) quá nhỏ (dưới 1s). Vui lòng đảm bảo file .env sử dụng suffix (ví dụ: '900s', '15m') thay vì số nguyên đơn thuần")
	}

	return cfg, nil
}
