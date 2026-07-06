package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/quocdev03/user-access-management/internal/constant"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/quocdev03/user-access-management/pkg/hash"
	"github.com/quocdev03/user-access-management/pkg/jwt"
)

const (
	maxFailedAttempts = 5
	lockDuration      = 30 * time.Minute
	otpExpiry         = 5 * time.Minute
)

type AuthService struct {
	userRepo     *repository.UserRepository
	otpService   *OTPService
	roleRepo     *repository.RoleRepository
	sessionRepo  *repository.SessionRepository
	auditLogRepo *repository.AuditLogRepository
	txManager    *database.TxManager
	cfg          *config.Config
	logger       *zap.Logger
}

func NewAuthService(
	userRepo *repository.UserRepository,
	otpService *OTPService,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	auditLogRepo *repository.AuditLogRepository,
	txManager *database.TxManager,
	cfg *config.Config,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		otpService:   otpService,
		roleRepo:     roleRepo,
		sessionRepo:  sessionRepo,
		auditLogRepo: auditLogRepo,
		txManager:    txManager,
		cfg:          cfg,
		logger:       logger,
	}
}

func (s *AuthService) generateTokenPair(userID uint64, roles []string) (string, string, error) {
	accessToken, _, err := jwt.GenerateToken(
		userID,
		roles,
		constant.TokenTypeAccess,
		s.cfg.JWT.AccessExpiry,
		s.cfg.JWT.Secret,
	)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err := jwt.GenerateToken(
		userID,
		roles,
		constant.TokenTypeRefresh,
		s.cfg.JWT.RefreshExpiry,
		s.cfg.JWT.Secret,
	)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	if len(req.Password) > 72 {
		return nil, apperror.ErrValidationError.WithMessage("Mật khẩu không được vượt quá 72 ký tự")
	}

	if err := hash.ValidatePasswordComplexity(req.Password); err != nil {
		return nil, apperror.ErrValidationError.WithMessage(err.Error())
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, apperror.ErrInvalidDateFormat
	}


	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash.HashPassword: %w", err)
	}

	user := &model.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		FullName:      req.FullName,
		Phone:         req.Phone,
		DateOfBirth:   dob,
		Status:        constant.StatusInactive,
		EmailVerified: false,
	}

	var sendEmail func()
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Create(txCtx, user); err != nil {
			if errors.Is(err, apperror.ErrConflict) {
				return apperror.ErrConflict
			}
			return fmt.Errorf("userRepo.Create: %w", err)
		}

		role, err := s.roleRepo.FindByName(txCtx, constant.RoleUser)
		if err != nil {
			return fmt.Errorf("roleRepo.FindByName: %w", err)
		}
		if role == nil {
			return fmt.Errorf("default role 'user' not found")
		}
		if err := s.roleRepo.AssignRoleToUser(txCtx, user.ID, role.ID); err != nil {
			return fmt.Errorf("roleRepo.AssignRoleToUser: %w", err)
		}

		fn, err := s.otpService.CreateAndSendOTP(
			txCtx,
			user.ID,
			user.Email,
			constant.OTPTypeEmailVerification,
			time.Now().UTC().Add(otpExpiry),
		)
		if err != nil {
			return err
		}
		sendEmail = fn
		return nil
	})

	if err != nil {
		return nil, err
	}

	go sendEmail()

	return &dto.RegisterResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   string(user.Status),
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error {

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		_ = hash.CheckPassword("dummy", "$2a$10$wK1WwzT8rVd3/xJ.8fU8K.R1.D.Fw1N98XzL3U.FwzT8rVd3/xJ.O")
		return apperror.ErrEmailNotFound
	}

	if user.EmailVerified {
		return apperror.ErrAccountAlreadyVerified
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.otpService.VerifyOTP(txCtx, user.ID, constant.OTPTypeEmailVerification, req.OTP); err != nil {
			return err
		}

		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, user.ID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		userForUpdate.Status = constant.StatusActive
		userForUpdate.EmailVerified = true
		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
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
		s.logger.Info("Yêu cầu gửi lại OTP cho email không tồn tại", zap.String("email", req.Email))
		return nil
	}

	if user.EmailVerified {
		return apperror.ErrAccountAlreadyVerified
	}

	sendEmail, err := s.otpService.CreateAndSendOTP(
		ctx,
		user.ID,
		user.Email,
		constant.OTPTypeEmailVerification,
		time.Now().UTC().Add(otpExpiry),
	)
	if err != nil {
		return err
	}
	go sendEmail()

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
		_ = hash.CheckPassword("dummy", "$2a$10$wK1WwzT8rVd3/xJ.8fU8K.R1.D.Fw1N98XzL3U.FwzT8rVd3/xJ.O")
		return nil, apperror.ErrInvalidCredentials
	}

	if err := s.verifyUserStatus(ctx, user); err != nil {
		return nil, err
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		s.logAudit(ctx, user.ID, "LOGIN", ipAddress, userAgent, "failure")
		return nil, handleFailedLogin(ctx, s.userRepo, s.logger, user)
	}

	if user.MustChangePassword {
		s.logAudit(ctx, user.ID, "LOGIN", ipAddress, userAgent, "failure_must_change_password")
		return nil, apperror.ErrMustChangePassword
	}

	return s.grantTokensAndSession(ctx, user, ipAddress, userAgent)
}

func (s *AuthService) verifyUserStatus(ctx context.Context, user *model.User) error {
	if user.Status == constant.StatusLocked {
		if user.LockedUntil == nil || user.LockedUntil.After(time.Now().UTC()) {
			return apperror.ErrAccountLocked
		}
		if err := s.userRepo.UnlockIfExpired(ctx, user.ID); err != nil {
			s.logger.Error("Không thể reset trạng thái khóa của user trong database", zap.Error(err))
		}
		user.Status = constant.StatusActive
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
	}

	if user.Status == constant.StatusInactive {
		return apperror.ErrAccountInactive
	}
	return nil
}

func (s *AuthService) logAudit(ctx context.Context, userID uint64, action, ip, ua, status string) {
	if err := s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:    &userID,
		Action:    action,
		IPAddress: &ip,
		UserAgent: &ua,
		Status:    status,
	}); err != nil {
		s.logger.Warn("Không thể ghi audit log", zap.String("action", action), zap.Error(err))
	}
}

func (s *AuthService) grantTokensAndSession(ctx context.Context, user *model.User, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	var accessToken, refreshToken string
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, user.ID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		now := time.Now()
		userForUpdate.LastLoginAt = &now
		userForUpdate.FailedLoginAttempts = 0
		userForUpdate.Status = constant.StatusActive
		userForUpdate.LockedUntil = nil
		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}

		var roles []string
		roles, err = s.roleRepo.GetRolesByUserID(txCtx, user.ID)
		if err != nil {
			s.logger.Warn("Không lấy được roles của user", zap.Uint64("user_id", user.ID), zap.Error(err))
			roles = []string{constant.RoleUser}
		}

		accessToken, refreshToken, err = s.generateTokenPair(user.ID, roles)
		if err != nil {
			return err
		}

		sessionExpiresAt := time.Now().Add(s.cfg.JWT.RefreshExpiry)
		session := &model.Session{
			UserID:           user.ID,
			TokenHash:        hash.SHA256(accessToken),
			RefreshTokenHash: hash.SHA256(refreshToken),
			IPAddress:        &ipAddress,
			UserAgent:        &userAgent,
			ExpiresAt:        sessionExpiresAt,
		}

		if err := s.sessionRepo.Create(txCtx, session); err != nil {
			s.logger.Error("Không thể tạo session trong MySQL", zap.Error(err))
			return fmt.Errorf("failed to create session: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, user.ID, "LOGIN", ipAddress, userAgent, "success")

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

	if claims.Type != constant.TokenTypeRefresh {
		return nil, apperror.ErrNotRefreshToken
	}

	refreshHash := hash.SHA256(req.RefreshToken)
	var res *dto.RefreshTokenResponse

	isRevoked, err := s.sessionRepo.IsRefreshTokenRevoked(ctx, refreshHash)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.IsRefreshTokenRevoked: %w", err)
	}
	if isRevoked {
		s.logger.Warn("Phát hiện Token Reuse (Refresh Token bị đánh cắp)", zap.Uint64("user_id", claims.UserID))
		_ = s.sessionRepo.DeleteByUserID(ctx, claims.UserID)
		_ = s.sessionRepo.RevokeAllUserTokens(ctx, claims.UserID, s.cfg.JWT.RefreshExpiry)
		return nil, apperror.ErrTokenReuse
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		session, err := s.sessionRepo.FindByRefreshTokenHashForUpdate(txCtx, refreshHash)
		if err != nil {
			return fmt.Errorf("sessionRepo.FindByRefreshTokenHashForUpdate: %w", err)
		}
		if session == nil {
			return apperror.ErrSessionExpired
		}
		if session.ExpiresAt.Before(time.Now()) {
			_ = s.sessionRepo.DeleteByRefreshTokenHash(txCtx, refreshHash)
			return apperror.ErrRefreshTokenInvalid
		}

		user, err := s.userRepo.FindByID(txCtx, claims.UserID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByID: %w", err)
		}
		if user == nil {
			_ = s.sessionRepo.DeleteByUserID(txCtx, claims.UserID)
			return apperror.ErrAccountDisabled
		}

		if err := s.verifyUserStatus(txCtx, user); err != nil {
			_ = s.sessionRepo.DeleteByUserID(txCtx, claims.UserID)
			return err
		}

		roles, err := s.roleRepo.GetRolesByUserID(txCtx, user.ID)
		if err != nil {
			roles = []string{constant.RoleUser}
		}

		newAccessToken, newRefreshToken, err := s.generateTokenPair(claims.UserID, roles)
		if err != nil {
			return err
		}

		session.TokenHash = hash.SHA256(newAccessToken)
		session.RefreshTokenHash = hash.SHA256(newRefreshToken)
		session.IPAddress = &ipAddress
		session.UserAgent = &userAgent
		session.ExpiresAt = time.Now().Add(s.cfg.JWT.RefreshExpiry)

		if err := s.sessionRepo.Update(txCtx, session); err != nil {
			s.logger.Error("Không thể cập nhật session khi refresh token", zap.Error(err))
			return fmt.Errorf("failed to update session: %w", err)
		}

		res = &dto.RefreshTokenResponse{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		if err := s.sessionRepo.AddRevokedRefreshToken(ctx, refreshHash, ttl); err != nil {
			s.logger.Warn("Không thể đưa refresh token cũ vào blacklist", zap.Error(err))
		}
	}

	return res, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string, claims *jwt.Claims) error {
	expTime := claims.ExpiresAt.Time
	ttl := time.Until(expTime)

	if ttl > 0 {
		if err := s.sessionRepo.AddToBlacklist(ctx, claims.ID, ttl); err != nil {
			s.logger.Warn("Không thể blacklist access token", zap.String("jti", claims.ID), zap.Error(err))
		}
	}

	tokenHash := hash.SHA256(rawToken)
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

func handleFailedLogin(ctx context.Context, userRepo *repository.UserRepository, logger *zap.Logger, user *model.User) error {
	attempts, _ := userRepo.IncrementFailedLogins(ctx, user.ID)

	if attempts >= maxFailedAttempts {
		lockedUntil := time.Now().UTC().Add(lockDuration)
		if err := userRepo.LockAccount(ctx, user.ID, lockedUntil, attempts); err != nil {
			logger.Error("Không thể khóa tài khoản", zap.Error(err))
			return apperror.ErrInvalidCredentials
		}

		logger.Warn("Tài khoản bị khóa do sai mật khẩu nhiều lần",
			zap.String("email", user.Email),
			zap.Int("failed_attempts", attempts),
		)
		return apperror.ErrAccountLocked.WithMessage(fmt.Sprintf("Tài khoản bị khóa %d phút do nhập sai mật khẩu %d lần", int(lockDuration.Minutes()), maxFailedAttempts))
	}
	return apperror.ErrInvalidCredentials
}

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
	if len([]byte(req.NewPassword)) > 72 {
		return apperror.ErrValidationError.WithMessage("Mật khẩu không được vượt quá 72 ký tự")
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
	if len([]byte(req.NewPassword)) > 72 {
		return apperror.ErrValidationError.WithMessage("Mật khẩu không được vượt quá 72 ký tự")
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
	if len([]byte(req.NewPassword)) > 72 {
		return apperror.ErrValidationError.WithMessage("Mật khẩu không được vượt quá 72 ký tự")
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

const maxOTPAttempts = 5

type OTPService struct {
	otpRepo     *repository.OTPRepository
	mailService *MailService
	logger      *zap.Logger
}

func NewOTPService(otpRepo *repository.OTPRepository, mailService *MailService, logger *zap.Logger) *OTPService {
	return &OTPService{
		otpRepo:     otpRepo,
		mailService: mailService,
		logger:      logger,
	}
}

func (s *OTPService) GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generateOTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *OTPService) CreateAndSendOTP(
	ctx context.Context,
	userID uint64,
	email string,
	otpType string,
	expiresAt time.Time,
) (func(), error) {
	otp, err := s.GenerateOTP()
	if err != nil {
		return nil, err
	}

	if err := s.otpRepo.Create(ctx, userID, otp, otpType, expiresAt); err != nil {
		return nil, fmt.Errorf("otpRepo.Create: %w", err)
	}

	sendFunc := func() {
		if err := s.mailService.SendVerificationEmail(email, otp); err != nil {
			s.logger.Error("Không thể gửi email OTP", zap.String("email", email), zap.Error(err))
		}
	}

	return sendFunc, nil
}

// VerifyOTP kiểm tra mã OTP đầu vào.
// Lưu ý: Nếu được gọi bên trong một database transaction, biến ctx truyền vào phải là txCtx
// để đảm bảo row level lock (FOR UPDATE) hoạt động đúng trên cùng một kết nối.
func (s *OTPService) VerifyOTP(ctx context.Context, userID uint64, otpType string, inputCode string) error {
	otpCode, err := s.otpRepo.GetLatestValidCodeForUpdate(ctx, userID, otpType)
	if err != nil {
		return fmt.Errorf("otpRepo.GetLatestValidCodeForUpdate: %w", err)
	}
	if otpCode == nil {
		return apperror.ErrOTPExpired
	}

	if otpCode.Attempts >= maxOTPAttempts {
		return apperror.ErrOTPMaxAttempts
	}

	if subtle.ConstantTimeCompare([]byte(otpCode.Code), []byte(inputCode)) != 1 {
		attempts, err := s.otpRepo.IncrementAttempts(ctx, otpCode.ID)
		if err != nil {
			return fmt.Errorf("otpRepo.IncrementAttempts: %w", err)
		}
		if attempts >= maxOTPAttempts {
			return apperror.ErrOTPMaxAttempts
		}
		return apperror.ErrOTPInvalid
	}

	if err := s.otpRepo.MarkAsUsed(ctx, otpCode.ID); err != nil {
		return fmt.Errorf("otpRepo.MarkAsUsed: %w", err)
	}

	return nil
}
