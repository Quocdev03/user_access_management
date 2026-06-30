package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New khởi tạo cấu hình zap.Logger tương ứng dựa trên môi trường chạy (development / production)
func New(env string) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return config.Build()
}
