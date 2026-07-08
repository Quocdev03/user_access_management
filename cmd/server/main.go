// Package main là điểm bắt đầu (entry point) cho User Access Management service.
// Nó thực hiện nạp cấu hình, thiết lập kết nối cơ sở dữ liệu (MySQL, Redis),
// khởi chạy background workers để dọn dẹp dữ liệu, cấu hình HTTP router,
// và quản lý quá trình tắt server an toàn (graceful shutdown).
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/internal/router"
	"github.com/quocdev03/user-access-management/internal/worker"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/quocdev03/user-access-management/pkg/logger"
	"go.uber.org/zap"
)

// @title User Access Management API
// @version 1.0
// @description Hệ thống quản lý truy cập người dùng RESTful API bằng Golang/Gin.
// @termsOfService http://swagger.io/terms/
// @contact.name Quoc Dev
// @contact.email quocdt2003@gmail.com
// @host localhost:8080
// @BasePath /api/v1
// main là hàm chính khởi động toàn bộ ứng dụng.
// Nó nạp biến môi trường, khởi tạo logger toàn cục, thiết lập
// các kết nối đến MySQL và Redis, sau đó gắn (mount) Gin HTTP router.
// Hàm này cũng lắng nghe các tín hiệu ngắt từ hệ điều hành (SIGINT, SIGTERM)
// để đảm bảo mọi tài nguyên và kết nối được đóng an toàn trước khi tiến trình kết thúc.
func main() {
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logr, err := logger.New(cfg.App.Env)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logr.Sync() }()

	zap.ReplaceGlobals(logr)

	logr.Info("Starting User Access Management Service...", zap.String("env", cfg.App.Env))

	db, err := database.ConnectMySQL(cfg.Database)
	if err != nil {
		logr.Fatal("Failed to connect to MySQL", zap.Error(err))
	}
	defer db.Close()
	logr.Info("Connected to MySQL successfully")

	if err := database.RunMigrations(cfg.Database, "migrations"); err != nil {
		logr.Fatal("Failed to run database migrations", zap.Error(err))
	}
	logr.Info("Database migrations applied successfully")

	redisClient, err := database.ConnectRedis(cfg.Redis)
	if err != nil {
		logr.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	logr.Info("Connected to Redis successfully")

	otpRepo := repository.NewOTPRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	passwordRepo := repository.NewPasswordRepository(db)
	cleanupWorker := worker.NewCleanupWorker(otpRepo, sessionRepo, passwordRepo, logr)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go cleanupWorker.Start(workerCtx, 24*time.Hour)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := router.Setup(db, redisClient, logr, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.App.Port),
		Handler: r,
	}

	go func() {
		logr.Info("Server listening", zap.String("port", cfg.App.Port), zap.String("url", fmt.Sprintf("http://localhost:%s", cfg.App.Port)))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logr.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logr.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logr.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logr.Info("Server exiting")
}
