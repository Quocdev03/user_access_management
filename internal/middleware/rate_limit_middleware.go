package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/response"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware giới hạn số lượng request từ một địa chỉ IP.
// Nếu vượt quá giới hạn (limit) trong khoảng thời gian (window), IP sẽ bị ban trong banDuration.
func RateLimitMiddleware(redisClient *redis.Client, limit int, window time.Duration, banDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()

		// 1. Kiểm tra xem IP có đang bị ban hay không
		banKey := fmt.Sprintf("ip_ban:%s", ip)
		isBanned, err := redisClient.Exists(ctx, banKey).Result()
		if err == nil && isBanned > 0 {
			response.Error(c, apperror.ErrIPBanned)
			c.Abort()
			return
		}

		// 2. Tính toán rate limit theo Fixed Window
		now := time.Now().Unix()
		windowSec := int64(window.Seconds())
		if windowSec <= 0 {
			windowSec = 60
		}
		bucket := now / windowSec
		limitKey := fmt.Sprintf("ratelimit:%s:%d", ip, bucket)

		// Sử dụng Redis TxPipeline để thực hiện atomic INCR và EXPIRE
		pipe := redisClient.TxPipeline()
		incr := pipe.Incr(ctx, limitKey)
		pipe.Expire(ctx, limitKey, window+10*time.Second) // Giữ key lâu hơn window một chút để tránh race
		_, err = pipe.Exec(ctx)

		if err != nil {
			// Fail-open: Nếu Redis lỗi, bỏ qua rate limit để không ảnh hưởng đến người dùng
			c.Next()
			return
		}

		count := incr.Val()
		if count > int64(limit) {
			// 3. Vượt quá giới hạn -> Thực hiện ban IP
			_ = redisClient.Set(ctx, banKey, "1", banDuration).Err()
			response.Error(c, apperror.ErrIPBanned)
			c.Abort()
			return
		}

		c.Next()
	}
}
