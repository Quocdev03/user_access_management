package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/quocdev03/user-access-management/internal/handler"
	"github.com/quocdev03/user-access-management/internal/middleware"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)


func Setup(db *sqlx.DB, redisClient *redis.Client, logger *zap.Logger, cfg *config.Config) *gin.Engine {
	r := gin.Default()


	r.Use(middleware.CORSMiddleware())


	userRepo := repository.NewUserRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	passwordRepo := repository.NewPasswordRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	mailService := service.NewMailService(cfg, logger)
	txManager := database.NewTxManager(db)
	
	authService := service.NewAuthService(userRepo, otpRepo, roleRepo, sessionRepo, auditLogRepo, mailService, txManager, cfg, logger)
	passwordService := service.NewPasswordService(userRepo, sessionRepo, passwordRepo, mailService, txManager, cfg, logger)

	authHandler := handler.NewAuthHandler(authService, passwordService)

	authMiddleware := middleware.AuthMiddleware(cfg, sessionRepo, logger)

	health := r.Group("/health")
	{
		health.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "UP"})
		})

		health.GET("/ready", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			// Ping DB
			if err := db.PingContext(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "MySQL down"})
				return
			}
			// Ping Redis
			if err := redisClient.Ping(ctx).Err(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "Redis down"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "UP", "mysql": "UP", "redis": "UP"})
		})
	}


	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-verification-email", authHandler.ResendVerificationEmail)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh-token", authHandler.RefreshToken)
			auth.POST("/logout", authMiddleware, authHandler.Logout)
			auth.POST("/logout-all", authMiddleware, authHandler.LogoutAll)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/change-password", authMiddleware, authHandler.ChangePassword)
		}
	}

	return r
}

