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
	"github.com/quocdev03/user-access-management/internal/router"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/quocdev03/user-access-management/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize Logger
	logr, err := logger.New(cfg.App.Env)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logr.Sync()
	
	zap.ReplaceGlobals(logr) // Replace global zap logger

	logr.Info("Starting User Access Management Service...", zap.String("env", cfg.App.Env))

	// 3. Connect to MySQL
	db, err := database.ConnectMySQL(cfg.Database)
	if err != nil {
		logr.Fatal("Failed to connect to MySQL", zap.Error(err))
	}
	defer db.Close()
	logr.Info("Connected to MySQL successfully")

	// 4. Connect to Redis
	redisClient, err := database.ConnectRedis(cfg.Redis)
	if err != nil {
		logr.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	logr.Info("Connected to Redis successfully")

	// 5. Setup Gin Router
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := router.Setup(db, redisClient, logr)

	// 6. Start HTTP Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.App.Port),
		Handler: r,
	}

	go func() {
		logr.Info("Server listening on port", zap.String("port", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logr.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logr.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logr.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logr.Info("Server exiting")
}
