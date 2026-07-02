package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/response"
	"github.com/redis/go-redis/v9"
)

func RateLimitMiddleware(redisClient *redis.Client, name string, limit int, banLimit int, window time.Duration, banDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()

		banKey := fmt.Sprintf("ip_ban:%s", ip)
		isBanned, err := redisClient.Exists(ctx, banKey).Result()
		if err == nil && isBanned > 0 {
			response.Error(c, apperror.ErrIPBanned)
			c.Abort()
			return
		}

		now := time.Now().Unix()
		windowSec := int64(window.Seconds())
		if windowSec <= 0 {
			windowSec = 60
		}
		bucket := now / windowSec
		limitKey := fmt.Sprintf("ratelimit:%s:%s:%d", name, ip, bucket)

		pipe := redisClient.TxPipeline()
		incr := pipe.Incr(ctx, limitKey)
		pipe.Expire(ctx, limitKey, window+10*time.Second)
		_, err = pipe.Exec(ctx)

		if err != nil {
			c.Next()
			return
		}

		count := incr.Val()
		if count > int64(banLimit) {
			_ = redisClient.Set(ctx, banKey, "1", banDuration).Err()
			response.Error(c, apperror.ErrIPBanned)
			c.Abort()
			return
		}

		if count > int64(limit) {
			response.Error(c, apperror.ErrRateLimited)
			c.Abort()
			return
		}

		c.Next()
	}
}
