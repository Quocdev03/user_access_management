package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/quocdev03/user-access-management/pkg/hash"
	"github.com/quocdev03/user-access-management/pkg/jwt"
)

// Hằng số nghiệp vụ (UC-06)
const (
	maxFailedAttempts = 5
	lockDuration      = 30 * time.Minute
	otpExpiry         = 5 * time.Minute
	maxOTPAttempts    = 5
	// Hash bcrypt cố định của chuỗi "dummy" để chống timing attack
	dummyBcryptHash   = "$2a$10$8K1p/a0fsBigaZE0N6cOG.e4s/8sYy1QyYtH4Yk9Y5UvI.G/k8M42"
)

type AuthService struct {
	userRepo          *repository.UserRepository
	otpRepo           *repository.OTPRepository
	roleRepo          *repository.RoleRepository
	sessionRepo       *repository.SessionRepository
	auditLogRepo      *repository.AuditLogRepository
	mailService       *MailService
	txManager         *database.TxManager
	cfg               *config.Config
	logger            *zap.Logger
}

func NewAuthService(
	userRepo *repository.UserRepository,
	otpRepo *repository.OTPRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	auditLogRepo *repository.AuditLogRepository,
	mailService *MailService,
	txManager *database.TxManager,
	cfg *config.Config,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		otpRepo:      otpRepo,
		roleRepo:     roleRepo,
		sessionRepo:  sessionRepo,
		auditLogRepo: auditLogRepo,
		mailService:  mailService,
		txManager:    txManager,
		cfg:          cfg,
		logger:       logger,
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

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

// generateTokenPair sinh cặp Access và Refresh token
func (s *AuthService) generateTokenPair(userID uint64, roles []string) (string, string, error) {
	accessToken, _, err := jwt.GenerateToken(userID, roles, "access", s.cfg.JWT.AccessExpiry, s.cfg.JWT.Secret)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err := jwt.GenerateToken(userID, roles, "refresh", s.cfg.JWT.RefreshExpiry, s.cfg.JWT.Secret)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash.HashPassword: %w", err)
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, apperror.ErrInvalidDateFormat
	}

	user := &model.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		FullName:      req.FullName,
		Phone:         req.Phone,
		DateOfBirth:   dob,
		Status:        model.StatusInactive,
		EmailVerified: false,
	}

	otp, _ := generateOTP()
	expiresAt := time.Now().Add(otpExpiry)

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Create(txCtx, user); err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				return apperror.ErrConflict
			}
			return fmt.Errorf("userRepo.Create: %w", err)
		}

		role, err := s.roleRepo.FindByName(txCtx, "user")
		if err != nil || role == nil {
			s.logger.Warn("Không tìm được default role", zap.Error(err))
		} else if err := s.roleRepo.AssignRoleToUser(txCtx, user.ID, role.ID); err != nil {
			s.logger.Warn("Không gán được role cho user", zap.Uint64("user_id", user.ID), zap.Error(err))
		}

		if err := s.otpRepo.Create(txCtx, user.ID, otp, "email_verification", expiresAt); err != nil {
			return fmt.Errorf("otpRepo.Create: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := s.mailService.SendVerificationEmail(user.Email, otp); err != nil {
		s.logger.Error("Không thể gửi email OTP đăng ký", zap.String("email", user.Email), zap.Error(err))
	}

	return &dto.RegisterResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	if user.EmailVerified {
		return apperror.ErrAccountAlreadyVerified
	}

	otpCode, err := s.otpRepo.GetLatestValidCode(ctx, user.ID, "email_verification")
	if err != nil {
		return fmt.Errorf("otpRepo.GetLatestValidCode: %w", err)
	}
	if otpCode == nil {
		return apperror.ErrOTPExpired
	}

	if otpCode.Attempts >= maxOTPAttempts {
		return apperror.ErrOTPMaxAttempts
	}

	if subtle.ConstantTimeCompare([]byte(otpCode.Code), []byte(req.OTP)) != 1 {
		attempts, _ := s.otpRepo.IncrementAttempts(ctx, otpCode.ID)
		if attempts >= maxOTPAttempts {
			return apperror.ErrOTPMaxAttempts
		}
		return apperror.ErrOTPInvalid
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.otpRepo.MarkAsUsed(txCtx, otpCode.ID); err != nil {
			return fmt.Errorf("otpRepo.MarkAsUsed: %w", err)
		}

		user.Status = model.StatusActive
		user.EmailVerified = true
		if err := s.userRepo.UpdateUser(txCtx, user); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	s.logger.Info("Xác thực email thành công", zap.String("email", user.Email))
	return nil
}

func (s *AuthService) ResendVerificationEmail(ctx context.Context, req dto.ResendVerificationEmailRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		return apperror.ErrEmailNotFound
	}

	if user.EmailVerified {
		return apperror.ErrAccountAlreadyVerified
	}

	otp, err := generateOTP()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(otpExpiry)

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.otpRepo.Create(txCtx, user.ID, otp, "email_verification", expiresAt); err != nil {
			return fmt.Errorf("otpRepo.Create: %w", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if err := s.mailService.SendVerificationEmail(user.Email, otp); err != nil {
		s.logger.Error("Không thể gửi email OTP (Resend)", zap.String("email", user.Email), zap.Error(err))
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.Info("Đã gửi lại email xác thực", zap.String("email", user.Email))
	return nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	if len(ipAddress) > 45 {
		ipAddress = ipAddress[:45]
	}
	if len(userAgent) > 500 {
		userAgent = userAgent[:500]
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		_ = hash.CheckPassword(req.Password, dummyBcryptHash)
		return nil, apperror.ErrInvalidCredentials
	}

	if user.Status == model.StatusLocked {
		if user.LockedUntil == nil || user.LockedUntil.After(time.Now()) {
			return nil, apperror.ErrAccountLocked
		}
		user.Status = model.StatusActive
		user.FailedLoginAttempts = 0
	}

	if user.Status == model.StatusInactive {
		return nil, apperror.ErrAccountInactive
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		_ = s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID:    &user.ID,
			Action:    "LOGIN",
			IPAddress: &ipAddress,
			UserAgent: &userAgent,
			Status:    "failure",
		})
		return nil, handleFailedLogin(ctx, s.userRepo, s.logger, user)
	}

	now := time.Now()
	user.LastLoginAt = &now
	user.FailedLoginAttempts = 0
	user.Status = model.StatusActive
	user.LockedUntil = nil
	_ = s.userRepo.UpdateUser(ctx, user)

	roles, err := s.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		s.logger.Warn("Không lấy được roles của user", zap.Uint64("user_id", user.ID), zap.Error(err))
		roles = []string{"user"}
	}

	accessToken, refreshToken, err := s.generateTokenPair(user.ID, roles)
	if err != nil {
		return nil, err
	}

	sessionExpiresAt := time.Now().Add(s.cfg.JWT.RefreshExpiry)
	session := &model.Session{
		UserID:           user.ID,
		TokenHash:        hashToken(accessToken),
		RefreshTokenHash: hashToken(refreshToken),
		IPAddress:        &ipAddress,
		UserAgent:        &userAgent,
		ExpiresAt:        sessionExpiresAt,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		s.logger.Error("Không thể tạo session trong MySQL", zap.Error(err))
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserInfoResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshTokenRequest, ipAddress, userAgent string) (*dto.RefreshTokenResponse, error) {
	if len(ipAddress) > 45 {
		ipAddress = ipAddress[:45]
	}
	if len(userAgent) > 500 {
		userAgent = userAgent[:500]
	}

	claims, err := jwt.ParseToken(req.RefreshToken, s.cfg.JWT.Secret)
	if err != nil {
		return nil, apperror.ErrRefreshTokenInvalid
	}

	if claims.Type != "refresh" {
		return nil, apperror.ErrNotRefreshToken
	}

	refreshHash := hashToken(req.RefreshToken)

	isRevoked, _ := s.sessionRepo.IsRefreshTokenRevoked(ctx, refreshHash)
	if isRevoked {
		s.logger.Warn("Phát hiện Token Reuse (Refresh Token bị đánh cắp)", zap.Uint64("user_id", claims.UserID))
		_ = s.sessionRepo.DeleteByUserID(ctx, claims.UserID)
		_ = s.sessionRepo.RevokeAllUserTokens(ctx, claims.UserID, s.cfg.JWT.RefreshExpiry)
		return nil, apperror.ErrTokenReuse
	}

	session, err := s.sessionRepo.FindByRefreshTokenHash(ctx, refreshHash)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.FindByRefreshTokenHash: %w", err)
	}

	if session == nil {
		return nil, apperror.ErrSessionExpired
	}

	if session.ExpiresAt.Before(time.Now()) {
		_ = s.sessionRepo.DeleteByRefreshTokenHash(ctx, refreshHash)
		return nil, apperror.ErrRefreshTokenInvalid
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil || user == nil || user.Status != model.StatusActive {
		_ = s.sessionRepo.DeleteByUserID(ctx, claims.UserID)
		return nil, apperror.ErrAccountDisabled
	}

	roles, err := s.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		roles = []string{"user"}
	}

	newAccessToken, newRefreshToken, err := s.generateTokenPair(claims.UserID, roles)
	if err != nil {
		return nil, err
	}

	sessionExpiresAt := time.Now().Add(s.cfg.JWT.RefreshExpiry)
	session.TokenHash = hashToken(newAccessToken)
	session.RefreshTokenHash = hashToken(newRefreshToken)
	session.IPAddress = &ipAddress
	session.UserAgent = &userAgent
	session.ExpiresAt = sessionExpiresAt

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		s.logger.Error("Không thể cập nhật session khi refresh token", zap.Error(err))
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		_ = s.sessionRepo.AddRevokedRefreshToken(ctx, refreshHash, ttl)
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string, claims *jwt.Claims) error {
	expTime := claims.ExpiresAt.Time
	ttl := time.Until(expTime)

	if ttl > 0 {
		if err := s.sessionRepo.AddToBlacklist(ctx, claims.ID, ttl); err != nil {
			s.logger.Warn("Không thể blacklist access token", zap.String("jti", claims.ID), zap.Error(err))
		}
	}

	tokenHash := hashToken(rawToken)
	if err := s.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("sessionRepo.DeleteByTokenHash: %w", err)
	}

	s.logger.Info("Đăng xuất thành công", zap.Uint64("user_id", claims.UserID))
	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uint64) error {
	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		s.logger.Error("Không thể thu hồi session khi LogoutAll", zap.Error(err))
		return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
	}

	_ = s.sessionRepo.RevokeAllUserTokens(ctx, userID, s.cfg.JWT.RefreshExpiry)

	s.logger.Info("Đã đăng xuất khỏi tất cả thiết bị", zap.Uint64("user_id", userID))
	return nil
}

// handleFailedLogin là hàm chia sẻ để xử lý logic tăng số lần sai và khóa tài khoản
func handleFailedLogin(ctx context.Context, userRepo *repository.UserRepository, logger *zap.Logger, user *model.User) error {
	attempts, _ := userRepo.IncrementFailedLogins(ctx, user.ID)
	
	if attempts >= maxFailedAttempts {
		lockedUntil := time.Now().Add(lockDuration)
		user.Status = model.StatusLocked
		user.LockedUntil = &lockedUntil
		_ = userRepo.UpdateUser(ctx, user)
		
		logger.Warn("Tài khoản bị khóa do sai mật khẩu nhiều lần",
			zap.String("email", user.Email),
			zap.Int("failed_attempts", attempts),
		)
		return apperror.ErrAccountLocked.WithMessage(fmt.Sprintf("Tài khoản bị khóa %d phút do nhập sai mật khẩu %d lần", int(lockDuration.Minutes()), maxFailedAttempts))
	}
	return apperror.ErrInvalidCredentials
}
