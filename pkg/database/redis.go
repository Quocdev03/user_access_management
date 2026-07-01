package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/redis/go-redis/v9"
)

// ConnectRedis khởi tạo một kết nối tới máy chủ cơ sở dữ liệu Redis
func ConnectRedis(cfg config.RedisConfig) (*redis.Client, error) {
	var opt *redis.Options
	var err error

	// Hỗ trợ định dạng URL (vd: redis://red-xxxx:6379) từ các PaaS như Render
	if strings.HasPrefix(cfg.Host, "redis://") || strings.HasPrefix(cfg.Host, "rediss://") {
		opt, err = redis.ParseURL(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis url: %w", err)
		}
	} else {
		opt = &redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Password: cfg.Password, // mật khẩu kết nối Redis (nếu có)
			DB:       0,            // sử dụng cơ sở dữ liệu mặc định (DB 0)
			PoolSize: 100,
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
