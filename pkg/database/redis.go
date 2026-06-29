package database

import (
	"context"
	"fmt"
	"time"

	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/redis/go-redis/v9"
)

// ConnectRedis khởi tạo một kết nối tới máy chủ cơ sở dữ liệu Redis
func ConnectRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password, // mật khẩu kết nối Redis (nếu có)
		DB:       0,            // sử dụng cơ sở dữ liệu mặc định (DB 0)
		PoolSize: 100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
