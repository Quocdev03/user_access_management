package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/handler"
	"github.com/quocdev03/user-access-management/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func setupAuthRoutes(rg *gin.RouterGroup, authHandler *handler.AuthHandler, authMiddleware gin.HandlerFunc, redisClient *redis.Client) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", middleware.RateLimitMiddleware(redisClient, "register", 10, 30, time.Minute, 15*time.Minute), authHandler.Register)
		auth.POST("/verify-email", middleware.RateLimitMiddleware(redisClient, "verify-email", 10, 30, time.Minute, 15*time.Minute), authHandler.VerifyEmail)
		auth.POST("/resend-verification-email", middleware.RateLimitMiddleware(redisClient, "resend-verification-email", 5, 20, time.Minute, 15*time.Minute), authHandler.ResendVerificationEmail)

		auth.POST("/login", middleware.RateLimitMiddleware(redisClient, "login", 15, 50, time.Minute, 15*time.Minute), authHandler.Login)
		auth.POST("/refresh-token", middleware.RateLimitMiddleware(redisClient, "refresh-token", 30, 60, time.Minute, 15*time.Minute), authHandler.RefreshToken)
		auth.POST("/logout", authMiddleware, authHandler.Logout)
		auth.POST("/logout-all", authMiddleware, authHandler.LogoutAll)

		auth.POST("/forgot-password", middleware.RateLimitMiddleware(redisClient, "forgot-password", 5, 20, time.Minute, 15*time.Minute), authHandler.ForgotPassword)
		auth.POST("/reset-password", middleware.RateLimitMiddleware(redisClient, "reset-password", 10, 30, time.Minute, 15*time.Minute), authHandler.ResetPassword)
		auth.POST("/change-password", authMiddleware, middleware.RateLimitMiddleware(redisClient, "change-password", 10, 30, time.Minute, 15*time.Minute), authHandler.ChangePassword)
	}
}
