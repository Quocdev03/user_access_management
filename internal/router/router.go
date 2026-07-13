package router

import (
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

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/quocdev03/user-access-management/docs"
)

func Setup(db *sqlx.DB, redisClient *redis.Client, logger *zap.Logger, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	if len(cfg.App.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.App.TrustedProxies)
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(middleware.CORSMiddleware(cfg))

	userRepo := repository.NewUserRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	passwordRepo := repository.NewPasswordRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	mailService := service.NewMailService(cfg, logger)
	txManager := database.NewTxManager(db)

	otpService := service.NewOTPService(otpRepo, mailService, logger, cfg)

	authService := service.NewAuthService(userRepo, otpService, roleRepo, sessionRepo, deviceRepo, auditLogRepo, txManager, cfg, logger)
	passwordService := service.NewPasswordService(userRepo, sessionRepo, passwordRepo, mailService, txManager, cfg, logger)
	userService := service.NewUserService(
		userRepo,
		otpService,
		roleRepo,
		sessionRepo,
		deviceRepo,
		auditLogRepo,
		mailService,
		txManager,
		cfg,
		logger,
	)
	adminUserService := service.NewAdminUserService(userRepo, roleRepo, sessionRepo, auditLogRepo, mailService, txManager, cfg, logger)
	adminRoleService := service.NewAdminRoleService(roleRepo, permissionRepo, sessionRepo, txManager, cfg, logger)
	adminAuditLogService := service.NewAdminAuditLogService(auditLogRepo, logger)

	authHandler := handler.NewAuthHandler(authService, passwordService)
	userHandler := handler.NewUserHandler(userService)
	adminHandler := handler.NewAdminHandler(adminUserService, adminRoleService, adminAuditLogService)

	authMiddleware := middleware.AuthMiddleware(cfg, sessionRepo, logger)

	setupHealthRoutes(r, db, redisClient)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.RateLimitMiddleware(redisClient, "global", cfg.Security.RateLimitRequests, cfg.Security.RateLimitRequests*3, cfg.Security.RateLimitWindow, 15*time.Minute))
	{
		setupAuthRoutes(v1, authHandler, authMiddleware, redisClient)
		setupUserRoutes(v1, userHandler, authMiddleware, redisClient)
		setupAdminRoutes(v1, adminHandler, authMiddleware, roleRepo)
	}
	r.Static("/uploads", "./uploads")
	r.StaticFile("/", "./ui_test/index.html")
	if cfg.App.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return r
}
