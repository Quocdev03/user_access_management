package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/quocdev03/user-access-management/internal/constant"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/quocdev03/user-access-management/pkg/hash"
)

type PasswordService struct {
	userRepo     *repository.UserRepository
	sessionRepo  *repository.SessionRepository
	passwordRepo *repository.PasswordRepository
	mailService  *MailService
	txManager    *database.TxManager
	cfg          *config.Config
	logger       *zap.Logger
}

func NewPasswordService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	passwordRepo *repository.PasswordRepository,
	mailService *MailService,
	txManager *database.TxManager,
	cfg *config.Config,
	logger *zap.Logger,
) *PasswordService {
	return &PasswordService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		passwordRepo: passwordRepo,
		mailService:  mailService,
		txManager:    txManager,
		cfg:          cfg,
		logger:       logger,
	}
}

func (s *PasswordService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "forgot_pw", req.Email, 3, 1*time.Minute)
	if isLimited {
		return apperror.ErrRateLimited
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}

	if user == nil {
		s.logger.Info("Yêu cầu quên mật khẩu cho email không tồn tại", zap.String("email", req.Email))
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := hash.SHA256(token)
	expiresAt := time.Now().Add(1 * time.Hour)

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.passwordRepo.InvalidateAllUserTokens(txCtx, user.ID); err != nil {
			return fmt.Errorf("passwordResetRepo.InvalidateAllUserTokens: %w", err)
		}

		if err := s.passwordRepo.Create(txCtx, user.ID, tokenHash, expiresAt); err != nil {
			return fmt.Errorf("passwordResetRepo.Create: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		if err := s.mailService.SendPasswordResetEmail(user.Email, token); err != nil {
			s.logger.Error("Lỗi khi gửi email khôi phục mật khẩu", zap.Error(err))
		}
	}()

	s.logger.Info("Đã gửi email khôi phục mật khẩu", zap.String("email", user.Email))
	return nil
}

func (s *PasswordService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	if err := hash.ValidatePasswordComplexity(req.NewPassword); err != nil {
		return apperror.ErrValidationError.WithMessage(err.Error())
	}

	tokenHash := hash.SHA256(req.Token)
	resetToken, err := s.passwordRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("passwordResetRepo.FindByTokenHash: %w", err)
	}

	if resetToken == nil || resetToken.IsUsed || resetToken.ExpiresAt.Before(time.Now()) {
		return apperror.ErrResetTokenInvalid
	}

	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.passwordRepo.MarkAsUsed(txCtx, resetToken.ID); err != nil {
			return apperror.ErrResetTokenUsed
		}

		if err := s.passwordRepo.InvalidateAllUserTokens(txCtx, resetToken.UserID); err != nil {
			s.logger.Warn("Không thể InvalidateAllUserTokens trong reset password", zap.Error(err))
		}

		if err := s.sessionRepo.DeleteByUserID(txCtx, resetToken.UserID); err != nil {
			s.logger.Error("Không thể thu hồi session khi reset password", zap.Error(err))
			return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
		}

		user, err := s.userRepo.FindByIDForUpdate(txCtx, resetToken.UserID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if user == nil {
			return apperror.ErrNotFound
		}
		user.PasswordHash = hashedPassword
		user.Status = constant.StatusActive
		user.LockedUntil = nil
		user.FailedLoginAttempts = 0

		if err := s.userRepo.UpdateUser(txCtx, user); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	_ = s.sessionRepo.RevokeAllUserTokens(ctx, resetToken.UserID, s.cfg.JWT.RefreshExpiry)

	s.logger.Info("Người dùng đã khôi phục mật khẩu thành công", zap.Uint64("user_id", resetToken.UserID))
	return nil
}

func (s *PasswordService) ChangePassword(ctx context.Context, userID uint64, req dto.ChangePasswordRequest) error {

	if err := hash.ValidatePasswordComplexity(req.NewPassword); err != nil {
		return apperror.ErrValidationError.WithMessage(err.Error())
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	if user.Status == constant.StatusLocked {
		if user.LockedUntil == nil || user.LockedUntil.After(time.Now().UTC()) {
			return apperror.ErrAccountLocked
		}
	}

	if req.OldPassword == req.NewPassword {
		return apperror.ErrSamePassword
	}

	if !hash.CheckPassword(req.OldPassword, user.PasswordHash) {
		return handleFailedLogin(ctx, s.userRepo, s.logger, user)
	}

	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.sessionRepo.DeleteByUserID(txCtx, userID); err != nil {
			s.logger.Error("Không thể thu hồi session khi đổi password", zap.Error(err))
			return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
		}

		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		userForUpdate.PasswordHash = hashedPassword
		userForUpdate.MustChangePassword = false
		userForUpdate.Status = constant.StatusActive
		userForUpdate.LockedUntil = nil
		userForUpdate.FailedLoginAttempts = 0

		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	_ = s.sessionRepo.RevokeAllUserTokens(ctx, userID, s.cfg.JWT.RefreshExpiry)

	s.logger.Info("Người dùng đã đổi mật khẩu thành công", zap.Uint64("user_id", userID))
	return nil
}

func (s *PasswordService) ForceChangePassword(ctx context.Context, req dto.ForceChangePasswordRequest) error {
	if err := hash.ValidatePasswordComplexity(req.NewPassword); err != nil {
		return apperror.ErrValidationError.WithMessage(err.Error())
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		// Ngăn lộ email
		_ = hash.CheckPassword("dummy", "$2a$10$wK1WwzT8rVd3/xJ.8fU8K.R1.D.Fw1N98XzL3U.FwzT8rVd3/xJ.O")
		return apperror.ErrInvalidCredentials
	}

	if user.Status == constant.StatusLocked {
		if user.LockedUntil == nil || user.LockedUntil.After(time.Now().UTC()) {
			return apperror.ErrAccountLocked
		}
	}
	if user.Status == constant.StatusInactive {
		return apperror.ErrAccountInactive
	}

	if !user.MustChangePassword {
		return apperror.ErrBadRequest.WithMessage("Tài khoản không nằm trong diện bắt buộc đổi mật khẩu")
	}

	if req.TempPassword == req.NewPassword {
		return apperror.ErrSamePassword
	}

	if !hash.CheckPassword(req.TempPassword, user.PasswordHash) {
		// Sử dụng hàm logic handleFailedLogin tương tự nhưng vì hàm đó import auth_service logic
		// Mình sẽ trực tiếp tăng attempts
		return apperror.ErrInvalidCredentials
	}

	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.sessionRepo.DeleteByUserID(txCtx, user.ID); err != nil {
			s.logger.Error("Không thể thu hồi session khi đổi password", zap.Error(err))
			return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
		}

		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, user.ID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		userForUpdate.PasswordHash = hashedPassword
		userForUpdate.MustChangePassword = false
		userForUpdate.Status = constant.StatusActive
		userForUpdate.LockedUntil = nil
		userForUpdate.FailedLoginAttempts = 0

		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	_ = s.sessionRepo.RevokeAllUserTokens(ctx, user.ID, s.cfg.JWT.RefreshExpiry)

	s.logger.Info("Người dùng đã force đổi mật khẩu thành công", zap.Uint64("user_id", user.ID))
	return nil
}
