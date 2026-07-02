package worker

import (
	"context"
	"time"

	"github.com/quocdev03/user-access-management/internal/repository"
	"go.uber.org/zap"
)

type CleanupWorker struct {
	otpRepo      *repository.OTPRepository
	sessionRepo  *repository.SessionRepository
	passwordRepo *repository.PasswordRepository
	logger       *zap.Logger
}

func NewCleanupWorker(
	otpRepo *repository.OTPRepository,
	sessionRepo *repository.SessionRepository,
	passwordRepo *repository.PasswordRepository,
	logger *zap.Logger,
) *CleanupWorker {
	return &CleanupWorker{
		otpRepo:      otpRepo,
		sessionRepo:  sessionRepo,
		passwordRepo: passwordRepo,
		logger:       logger,
	}
}

func (w *CleanupWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.logger.Info("Cleanup worker started", zap.String("interval", interval.String()))

	// Chạy ngay 1 lần lúc startup (tuỳ chọn)
	w.cleanup(ctx)

	for {
		select {
		case <-ticker.C:
			w.cleanup(ctx)
		case <-ctx.Done():
			w.logger.Info("Cleanup worker stopped")
			return
		}
	}
}

func (w *CleanupWorker) cleanup(ctx context.Context) {
	w.logger.Info("Running cleanup tasks for expired records...")

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	now := time.Now().UTC()

	if err := w.otpRepo.DeleteExpired(timeoutCtx, now); err != nil {
		w.logger.Error("Lỗi khi xoá OTP cũ", zap.Error(err))
	}

	if err := w.sessionRepo.DeleteExpired(timeoutCtx, now); err != nil {
		w.logger.Error("Lỗi khi xoá session cũ", zap.Error(err))
	}

	if err := w.passwordRepo.DeleteExpired(timeoutCtx, now); err != nil {
		w.logger.Error("Lỗi khi xoá password reset token cũ", zap.Error(err))
	}

	w.logger.Info("Cleanup tasks finished")
}
