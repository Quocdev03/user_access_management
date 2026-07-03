package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/handler"
	"github.com/quocdev03/user-access-management/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func setupUserRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler, authMiddleware gin.HandlerFunc, redisClient *redis.Client) {
	users := rg.Group("/users", authMiddleware)
	{
		users.GET("/me", userHandler.GetProfile)
		users.PATCH("/me", userHandler.UpdateProfile)
		emailRL := middleware.RateLimitMiddleware(redisClient, "change-email", 5, 20, time.Minute, 15*time.Minute)

		users.POST("/me/email/request-change", emailRL, userHandler.RequestEmailChange)
		users.POST("/me/email/verify", emailRL, userHandler.VerifyEmailChange)
		users.POST("/me/email/resend-otp", emailRL, userHandler.ResendChangeEmailOTP)

		users.POST("/me/avatar", middleware.RateLimitMiddleware(redisClient, "upload-avatar", 5, 20, time.Minute, 15*time.Minute), userHandler.UploadAvatar)

		users.DELETE("/me/avatar", userHandler.DeleteAvatar)
	}
}
