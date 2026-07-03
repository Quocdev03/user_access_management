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
	r := gin.New()
	r.Use(gin.Recovery())
	// Custom zap logger middleware if needed, but for now simple setup
	r.Use(gin.Logger())

	r.Use(middleware.CORSMiddleware(cfg))

	r.StaticFile("/", "./ui_test/index.html")

	userRepo := repository.NewUserRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	passwordRepo := repository.NewPasswordRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	mailService := service.NewMailService(cfg, logger)
	txManager := database.NewTxManager(db)

	otpService := service.NewOTPService(otpRepo, mailService, logger)

	authService := service.NewAuthService(userRepo, otpService, roleRepo, sessionRepo, auditLogRepo, txManager, cfg, logger)
	passwordService := service.NewPasswordService(userRepo, sessionRepo, passwordRepo, mailService, txManager, cfg, logger)
	userService := service.NewUserService(
		userRepo,
		otpService,
		roleRepo,
		sessionRepo,
		mailService,
		txManager,
		cfg,
		logger,
	)

	authHandler := handler.NewAuthHandler(authService, passwordService)
	userHandler := handler.NewUserHandler(userService)

	authMiddleware := middleware.AuthMiddleware(cfg, sessionRepo, logger)

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

	v1 := r.Group("/api/v1")
	v1.Use(middleware.RateLimitMiddleware(redisClient, "global", cfg.Security.RateLimitRequests, cfg.Security.RateLimitRequests*3, cfg.Security.RateLimitWindow, 15*time.Minute))
	{
		setupAuthRoutes(v1, authHandler, authMiddleware, redisClient)
		setupUserRoutes(v1, userHandler, authMiddleware, redisClient)
	}

	return r
}
