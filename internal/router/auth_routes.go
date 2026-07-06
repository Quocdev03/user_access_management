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
		auth.POST("/login", middleware.RateLimitMiddleware(redisClient, "login", 15, 50, time.Minute, 15*time.Minute), authHandler.Login)
		auth.POST("/logout", authMiddleware, authHandler.Logout)
		auth.POST("/logout-all", authMiddleware, authHandler.LogoutAll)

		email := auth.Group("/email")
		{
			email.POST("/verify", middleware.RateLimitMiddleware(redisClient, "verify-email", 10, 30, time.Minute, 15*time.Minute), authHandler.VerifyEmail)
			email.POST("/resend", middleware.RateLimitMiddleware(redisClient, "resend-verification-email", 10, 30, time.Minute, 15*time.Minute), authHandler.ResendVerificationEmail)
		}

		token := auth.Group("/token")
		{
			token.POST("/refresh", middleware.RateLimitMiddleware(redisClient, "refresh-token", 30, 60, time.Minute, 15*time.Minute), authHandler.RefreshToken)
		}

		password := auth.Group("/password")
		{
			password.POST("/forgot", middleware.RateLimitMiddleware(redisClient, "forgot-password", 10, 30, time.Minute, 15*time.Minute), authHandler.ForgotPassword)
			password.POST("/reset", middleware.RateLimitMiddleware(redisClient, "reset-password", 10, 30, time.Minute, 15*time.Minute), authHandler.ResetPassword)
			password.POST("/change", authMiddleware, middleware.RateLimitMiddleware(redisClient, "change-password", 10, 30, time.Minute, 15*time.Minute), authHandler.ChangePassword)
			password.POST("/force-change", middleware.RateLimitMiddleware(redisClient, "force-change-password", 10, 30, time.Minute, 15*time.Minute), authHandler.ForceChangePassword)
		}
	}
}
