package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func setupHealthRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	health := r.Group("/health")
	{
		health.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "UP"})
		})

		health.GET("/ready", middleware.RateLimitMiddleware(redisClient, "ready", 10, 30, time.Minute, 15*time.Minute), func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			if err := db.PingContext(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "MySQL down"})
				return
			}
			if err := redisClient.Ping(ctx).Err(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "Redis down"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "UP", "mysql": "UP", "redis": "UP"})
		})
	}
}
