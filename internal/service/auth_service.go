package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
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
	dummyBcryptHash   = "$2a$10$8K1p/a0fsBigaZE0N6cOG.e4s/8sYy1QyYtH4Yk9Y5UvI.G/k8M42"
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
	accessToken, _, err := jwt.GenerateToken(userID, roles, constant.TokenTypeAccess, s.cfg.JWT.AccessExpiry, s.cfg.JWT.Secret)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err := jwt.GenerateToken(userID, roles, constant.TokenTypeRefresh, s.cfg.JWT.RefreshExpiry, s.cfg.JWT.Secret)
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

	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "register_email", req.Email, 3, 1*time.Minute)
	if isLimited {
		return nil, apperror.ErrRateLimited
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
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
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

		fn, err := s.otpService.CreateAndSendOTP(txCtx, user.ID, user.Email, constant.OTPTypeEmailVerification, time.Now().UTC().Add(otpExpiry))
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
		Status:   user.Status,
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error {
	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "verify_otp", req.Email, 5, 1*time.Minute)
	if isLimited {
		return apperror.ErrRateLimited
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		return nil
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
	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "resend_otp", req.Email, 3, 1*time.Minute)
	if isLimited {
		return apperror.ErrRateLimited
	}

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

	sendEmail, err := s.otpService.CreateAndSendOTP(ctx, user.ID, user.Email, constant.OTPTypeEmailVerification, time.Now().UTC().Add(otpExpiry))
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

	isLimited, err := s.sessionRepo.IncrementRateLimit(ctx, "login_email", req.Email, 5, 1*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.IncrementRateLimit: %w", err)
	}
	if isLimited {
		return nil, apperror.ErrRateLimited
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		_ = hash.CheckPassword(req.Password, dummyBcryptHash)
		return nil, apperror.ErrInvalidCredentials
	}

	if err := s.verifyUserStatus(ctx, user); err != nil {
		return nil, err
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		s.logAudit(ctx, user.ID, "LOGIN", ipAddress, userAgent, "failure")
		return nil, handleFailedLogin(ctx, s.userRepo, s.logger, user)
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
		} else {
			user.Status = constant.StatusActive
			user.FailedLoginAttempts = 0
			user.LockedUntil = nil
		}
	}

	if user.Status == constant.StatusInactive {
		return apperror.ErrAccountInactive
	}
	return nil
}

func (s *AuthService) logAudit(ctx context.Context, userID uint64, action, ip, ua, status string) {
	_ = s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:    &userID,
		Action:    action,
		IPAddress: &ip,
		UserAgent: &ua,
		Status:    status,
	})
}

func (s *AuthService) grantTokensAndSession(ctx context.Context, user *model.User, ipAddress, userAgent string) (*dto.LoginResponse, error) {
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
		return nil
	})
	if err != nil {
		return nil, err
	}

	roles, err := s.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		s.logger.Warn("Không lấy được roles của user", zap.Uint64("user_id", user.ID), zap.Error(err))
		roles = []string{constant.RoleUser}
	}

	accessToken, refreshToken, err := s.generateTokenPair(user.ID, roles)
	if err != nil {
		return nil, err
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

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		s.logger.Error("Không thể tạo session trong MySQL", zap.Error(err))
		return nil, fmt.Errorf("failed to create session: %w", err)
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

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.checkTokenReuse(txCtx, claims.UserID, refreshHash); err != nil {
			return err
		}

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

		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			_ = s.sessionRepo.AddRevokedRefreshToken(txCtx, refreshHash, ttl)
		}

		res = &dto.RefreshTokenResponse{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		}
		return nil
	})

	return res, err
}

func (s *AuthService) checkTokenReuse(ctx context.Context, userID uint64, refreshHash string) error {
	isRevoked, err := s.sessionRepo.IsRefreshTokenRevoked(ctx, refreshHash)
	if err != nil {
		return fmt.Errorf("sessionRepo.IsRefreshTokenRevoked: %w", err)
	}
	if isRevoked {
		s.logger.Warn("Phát hiện Token Reuse (Refresh Token bị đánh cắp)", zap.Uint64("user_id", userID))
		_ = s.sessionRepo.DeleteByUserID(ctx, userID)
		_ = s.sessionRepo.RevokeAllUserTokens(ctx, userID, s.cfg.JWT.RefreshExpiry)
		return apperror.ErrTokenReuse
	}
	return nil
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
