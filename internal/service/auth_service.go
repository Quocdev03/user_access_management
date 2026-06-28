package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/hash"
)

// Hằng số nghiệp vụ (UC-06)
const (
	maxFailedAttempts = 5
	lockDuration      = 30 * time.Minute
	otpExpiry         = 5 * time.Minute
	maxOTPAttempts    = 5
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)
	VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
}

type authService struct {
	userRepo repository.UserRepository
	otpRepo  repository.OTPRepository
	roleRepo repository.RoleRepository
	logger   *zap.Logger
}

func NewAuthService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	roleRepo repository.RoleRepository,
	logger *zap.Logger,
) AuthService {
	return &authService{
		userRepo: userRepo,
		otpRepo:  otpRepo,
		roleRepo: roleRepo,
		logger:   logger,
	}
}

// generateOTP sinh OTP 6 chữ số bằng crypto/rand (B4 — bảo mật)
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generateOTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// A3: Kiểm tra email tồn tại
	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if existingUser != nil {
		return nil, apperror.ErrConflict
	}

	// A3: Kiểm tra username tồn tại
	existingByUsername, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByUsername: %w", err)
	}
	if existingByUsername != nil {
		return nil, apperror.ErrConflict
	}

	// Hash password
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash.HashPassword: %w", err)
	}

	// Tạo User (trạng thái inactive)
	user := &model.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		FullName:      req.FullName,
		Status:        model.StatusInactive,
		EmailVerified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("userRepo.Create: %w", err)
	}

	// Gán role 'user'
	role, err := s.roleRepo.FindByName(ctx, "user")
	if err != nil {
		s.logger.Warn("Không tìm được default role", zap.Error(err))
	} else if role != nil {
		if err := s.roleRepo.AssignRoleToUser(ctx, user.ID, role.ID); err != nil {
			s.logger.Warn("Không gán được role cho user", zap.Uint64("user_id", user.ID), zap.Error(err))
		}
	}

	// A4: Sinh OTP bằng crypto/rand
	otp, err := generateOTP()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(otpExpiry)

	if err := s.otpRepo.Create(ctx, user.ID, otp, "email_verification", expiresAt); err != nil {
		return nil, fmt.Errorf("otpRepo.Create: %w", err)
	}

	// B1: Dùng Zap Logger thay vì fmt.Printf
	s.logger.Info("[MOCK MAIL] OTP đăng ký",
		zap.String("email", user.Email),
		zap.String("otp", otp),
	)

	return &dto.RegisterResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	if user.EmailVerified {
		return apperror.NewAppError("ALREADY_VERIFIED", "Email đã được xác thực", 400)
	}

	// Lấy OTP hợp lệ
	otpCode, err := s.otpRepo.GetLatestValidCode(ctx, user.ID, "email_verification")
	if err != nil {
		return fmt.Errorf("otpRepo.GetLatestValidCode: %w", err)
	}
	if otpCode == nil {
		return apperror.NewAppError("OTP_EXPIRED", "Không tìm thấy mã OTP hợp lệ hoặc đã hết hạn", 400)
	}

	if otpCode.Attempts >= maxOTPAttempts {
		return apperror.NewAppError("OTP_MAX_ATTEMPTS", "Bạn đã nhập sai OTP quá 5 lần, vui lòng yêu cầu mã mới", 400)
	}

	if otpCode.Code != req.OTP {
		_ = s.otpRepo.IncrementAttempts(ctx, otpCode.ID)
		return apperror.NewAppError("OTP_INVALID", "Mã OTP không đúng", 400)
	}

	// Cập nhật trạng thái user
	if err := s.userRepo.UpdateStatus(ctx, user.ID, model.StatusActive, true); err != nil {
		return fmt.Errorf("userRepo.UpdateStatus: %w", err)
	}

	// Đánh dấu OTP đã dùng (giữ log, không xóa cứng)
	_ = s.otpRepo.MarkAsUsed(ctx, otpCode.ID)

	s.logger.Info("Xác thực email thành công", zap.String("email", user.Email))
	return nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		return nil, apperror.ErrInvalidCredentials
	}

	// Kiểm tra trạng thái locked (với tự động unlock nếu hết hạn)
	if user.Status == model.StatusLocked {
		if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
			return nil, apperror.ErrAccountLocked
		} else if user.LockedUntil != nil {
			// Hết thời gian khóa -> tự động mở khóa
			_ = s.userRepo.UpdateStatus(ctx, user.ID, model.StatusActive, user.EmailVerified)
			user.Status = model.StatusActive
			user.FailedLoginAttempts = 0
		} else {
			return nil, apperror.ErrAccountLocked
		}
	}

	if user.Status == model.StatusInactive {
		return nil, apperror.NewAppError("INACTIVE_ACCOUNT", "Vui lòng xác thực email trước khi đăng nhập", 403)
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		// A1: Kiểm tra ngưỡng TRƯỚC khi increment để tránh off-by-one
		if user.FailedLoginAttempts+1 >= maxFailedAttempts {
			lockedUntil := time.Now().Add(lockDuration) // A2: 30 phút theo UC-06
			_ = s.userRepo.LockAccount(ctx, user.ID, lockedUntil)
			s.logger.Warn("Tài khoản bị khóa do sai mật khẩu nhiều lần",
				zap.String("email", user.Email),
				zap.Int("failed_attempts", user.FailedLoginAttempts+1),
			)
			return nil, apperror.NewAppError("ACCOUNT_LOCKED",
				fmt.Sprintf("Tài khoản bị khóa %d phút do nhập sai mật khẩu %d lần",
					int(lockDuration.Minutes()), maxFailedAttempts),
				423,
			)
		}
		_ = s.userRepo.IncrementFailedLogins(ctx, user.ID)
		return nil, apperror.ErrInvalidCredentials
	}

	// Đăng nhập thành công -> update last_login và reset counter
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	s.logger.Info("Đăng nhập thành công", zap.String("email", user.Email), zap.Uint64("user_id", user.ID))

	// TODO: Tạo JWT Access Token & Refresh Token (task tiếp theo)
	accessToken := "mock_access_token_jwt"
	refreshToken := "mock_refresh_token_jwt"

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserInfoResponse{ // B3: Dùng named struct thay vì anonymous
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}
