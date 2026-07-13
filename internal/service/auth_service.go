package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ua-parser/uap-go/uaparser"
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

// uaParser: uap-go (regexes nhúng sẵn). Lazy init — parse chính xác hơn heuristic strings.Contains.
var uaParser = sync.OnceValue(func() *uaparser.Parser {
	return uaparser.NewFromSaved()
})

type AuthService struct {
	userRepo     *repository.UserRepository
	otpService   *OTPService
	roleRepo     *repository.RoleRepository
	sessionRepo  *repository.SessionRepository
	deviceRepo   *repository.DeviceRepository
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
	deviceRepo *repository.DeviceRepository,
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
		deviceRepo:   deviceRepo,
		auditLogRepo: auditLogRepo,
		txManager:    txManager,
		cfg:          cfg,
		logger:       logger,
	}
}

func clampRequestMeta(ipAddress, userAgent string) (string, string) {
	if len(ipAddress) > 45 {
		ipAddress = ipAddress[:45]
	}
	if len(userAgent) > 500 {
		userAgent = userAgent[:500]
	}
	return ipAddress, userAgent
}

// parseUserAgent dùng ua-parser/uap-go. Trả về os, browser, device_type (mobile|tablet|desktop).
func parseUserAgent(ua string) (osName, browser, deviceType string) {
	if strings.TrimSpace(ua) == "" {
		return "Unknown", "Unknown", "desktop"
	}
	c := uaParser().Parse(ua)

	osName = "Unknown"
	if c.Os != nil && c.Os.Family != "" {
		osName = c.Os.Family
		if c.Os.Major != "" {
			osName = c.Os.ToString() // e.g. "Windows 10"
		}
	}

	browser = "Unknown"
	if c.UserAgent != nil && c.UserAgent.Family != "" {
		browser = c.UserAgent.Family
		if c.UserAgent.Major != "" {
			browser = c.UserAgent.ToString() // e.g. "Chrome 149.0.0"
		}
	}

	deviceType = mapDeviceType(c)
	return osName, browser, deviceType
}

func mapDeviceType(c *uaparser.Client) string {
	if c == nil || c.Device == nil {
		return "desktop"
	}
	fam := strings.TrimSpace(c.Device.Family)
	if fam == "" || strings.EqualFold(fam, "Other") || strings.EqualFold(fam, "Spider") {
		return "desktop"
	}
	lower := strings.ToLower(fam)
	switch {
	case strings.Contains(lower, "ipad"),
		strings.Contains(lower, "tablet"),
		strings.Contains(lower, "kindle"),
		strings.Contains(lower, "playbook"):
		return "tablet"
	case strings.Contains(lower, "iphone"),
		strings.Contains(lower, "phone"),
		strings.Contains(lower, "mobile"),
		strings.Contains(lower, "android"),
		strings.Contains(lower, "ipod"),
		strings.Contains(lower, "nokia"),
		strings.Contains(lower, "blackberry"):
		return "mobile"
	default:
		// Hầu hết Family cụ thể (Samsung SM-…, Pixel…) là mobile.
		return "mobile"
	}
}

func (s *AuthService) generateTokenPair(userID uint64, roles []string) (accessToken, refreshToken, accessJTI string, err error) {
	accessToken, accessJTI, err = jwt.GenerateToken(
		userID,
		roles,
		constant.TokenTypeAccess,
		s.cfg.JWT.AccessExpiry,
		s.cfg.JWT.Secret,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err = jwt.GenerateToken(
		userID,
		roles,
		constant.TokenTypeRefresh,
		s.cfg.JWT.RefreshExpiry,
		s.cfg.JWT.Secret,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, accessJTI, nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	if err := hash.ValidateNewPassword(req.Password); err != nil {
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
			time.Now().UTC().Add(s.cfg.Security.OTPExpiry),
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
		_ = hash.CheckPassword("dummy-password", hash.DummyPasswordHash)
		// Không lộ email chưa đăng ký (giống forgot-password).
		return apperror.ErrOTPInvalid
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
		time.Now().UTC().Add(s.cfg.Security.OTPExpiry),
	)
	if err != nil {
		return err
	}
	go sendEmail()

	s.logger.Info("Đã gửi lại email xác thực", zap.String("email", user.Email))
	return nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	ipAddress, userAgent = clampRequestMeta(ipAddress, userAgent)

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		_ = hash.CheckPassword("dummy-password", hash.DummyPasswordHash)
		return nil, apperror.ErrInvalidCredentials
	}

	if err := s.verifyUserStatus(ctx, user); err != nil {
		return nil, err
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		s.logAudit(ctx, user.ID, "LOGIN", ipAddress, userAgent, "failure")
		return nil, handleFailedLogin(ctx, s.userRepo, s.logger, user, s.cfg.Security.MaxFailedAttempts, s.cfg.Security.LockDuration)
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
	resource := "auth"
	if err := s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:    &userID,
		Action:    action,
		Resource:  &resource,
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

		var accessJTI string
		accessToken, refreshToken, accessJTI, err = s.generateTokenPair(user.ID, roles)
		if err != nil {
			return err
		}

		var deviceID *uint64
		osName, browserName, deviceType := parseUserAgent(userAgent)
		device := &model.Device{
			UserID:     user.ID,
			IPAddress:  &ipAddress,
			OS:         &osName,
			Browser:    &browserName,
			DeviceType: &deviceType,
		}
		if userAgent != "" {
			ua := userAgent
			device.DeviceName = &ua
		}
		if err := s.deviceRepo.FindOrCreate(txCtx, device); err != nil {
			s.logger.Warn("Không thể ghi device tracking", zap.Error(err), zap.Uint64("user_id", user.ID))
		} else if device.ID > 0 {
			deviceID = &device.ID
		}

		jti := accessJTI
		sessionExpiresAt := time.Now().Add(s.cfg.JWT.RefreshExpiry)
		session := &model.Session{
			UserID:           user.ID,
			TokenHash:        hash.SHA256(accessToken),
			RefreshTokenHash: hash.SHA256(refreshToken),
			JTI:              &jti,
			IPAddress:        &ipAddress,
			UserAgent:        &userAgent,
			DeviceID:         deviceID,
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
	ipAddress, userAgent = clampRequestMeta(ipAddress, userAgent)

	claims, err := jwt.ParseToken(req.RefreshToken, s.cfg.JWT.Secret)
	if err != nil {
		return nil, apperror.ErrRefreshTokenInvalid
	}

	if claims.Type != constant.TokenTypeRefresh {
		return nil, apperror.ErrNotRefreshToken
	}

	refreshHash := hash.SHA256(req.RefreshToken)
	var res *dto.RefreshTokenResponse
	var oldAccessJTI string

	isRevoked, err := s.sessionRepo.IsRefreshTokenRevoked(ctx, refreshHash)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.IsRefreshTokenRevoked: %w", err)
	}
	if isRevoked {
		s.logger.Warn("Phát hiện Token Reuse (Refresh Token bị đánh cắp)", zap.Uint64("user_id", claims.UserID))
		_ = s.sessionRepo.InvalidateUserSessions(ctx, claims.UserID, s.cfg.JWT.RefreshExpiry)
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

		if session.JTI != nil {
			oldAccessJTI = *session.JTI
		}

		newAccessToken, newRefreshToken, newAccessJTI, err := s.generateTokenPair(claims.UserID, roles)
		if err != nil {
			return err
		}

		jti := newAccessJTI
		session.TokenHash = hash.SHA256(newAccessToken)
		session.RefreshTokenHash = hash.SHA256(newRefreshToken)
		session.JTI = &jti
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
	if oldAccessJTI != "" {
		if err := s.sessionRepo.AddToBlacklist(ctx, oldAccessJTI, s.cfg.JWT.AccessExpiry); err != nil {
			s.logger.Warn("Không thể blacklist access token cũ sau refresh", zap.String("jti", oldAccessJTI), zap.Error(err))
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
	if err := s.sessionRepo.InvalidateUserSessions(ctx, userID, s.cfg.JWT.RefreshExpiry); err != nil {
		s.logger.Error("Không thể thu hồi session khi LogoutAll", zap.Error(err))
		return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
	}

	s.logger.Info("Đã đăng xuất khỏi tất cả thiết bị", zap.Uint64("user_id", userID))
	return nil
}

func handleFailedLogin(
	ctx context.Context,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
	user *model.User,
	maxAttempts int,
	lockDur time.Duration,
) error {
	attempts, _ := userRepo.IncrementFailedLogins(ctx, user.ID)

	if attempts >= maxAttempts {
		lockedUntil := time.Now().UTC().Add(lockDur)
		if err := userRepo.LockAccount(ctx, user.ID, lockedUntil, attempts); err != nil {
			logger.Error("Không thể khóa tài khoản", zap.Error(err))
			return apperror.ErrInvalidCredentials
		}

		logger.Warn("Tài khoản bị khóa do sai mật khẩu nhiều lần",
			zap.String("email", user.Email),
			zap.Int("failed_attempts", attempts),
		)
		return apperror.ErrAccountLocked.WithMessage(fmt.Sprintf(
			"Tài khoản bị khóa %d phút do nhập sai mật khẩu %d lần",
			int(lockDur.Minutes()),
			maxAttempts,
		))
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
			s.logger.Warn("FALLBACK reset token (dev) — chỉ dùng khi SMTP lỗi",
				zap.String("email", user.Email),
				zap.String("token", token),
			)
		}
	}()

	s.logger.Info("Yêu cầu khôi phục mật khẩu đã xử lý (mail async)", zap.String("email", user.Email))
	return nil
}

func (s *PasswordService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	if err := hash.ValidateNewPassword(req.NewPassword); err != nil {
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

	if err := s.sessionRepo.InvalidateUserSessions(ctx, resetToken.UserID, s.cfg.JWT.RefreshExpiry); err != nil {
		s.logger.Warn("Không thể invalidate sessions sau reset password", zap.Error(err))
	}

	s.logger.Info("Người dùng đã khôi phục mật khẩu thành công", zap.Uint64("user_id", resetToken.UserID))
	return nil
}

func (s *PasswordService) ChangePassword(ctx context.Context, userID uint64, req dto.ChangePasswordRequest) error {
	if err := hash.ValidateNewPassword(req.NewPassword); err != nil {
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
		return handleFailedLogin(ctx, s.userRepo, s.logger, user, s.cfg.Security.MaxFailedAttempts, s.cfg.Security.LockDuration)
	}

	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
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

	if err := s.sessionRepo.InvalidateUserSessions(ctx, userID, s.cfg.JWT.RefreshExpiry); err != nil {
		s.logger.Warn("Không thể invalidate sessions sau change password", zap.Error(err))
	}

	s.logger.Info("Người dùng đã đổi mật khẩu thành công", zap.Uint64("user_id", userID))
	return nil
}

func (s *PasswordService) ForceChangePassword(ctx context.Context, req dto.ForceChangePasswordRequest) error {
	if err := hash.ValidateNewPassword(req.NewPassword); err != nil {
		return apperror.ErrValidationError.WithMessage(err.Error())
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		_ = hash.CheckPassword("dummy-password", hash.DummyPasswordHash)
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
		return handleFailedLogin(ctx, s.userRepo, s.logger, user, s.cfg.Security.MaxFailedAttempts, s.cfg.Security.LockDuration)
	}

	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
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

	if err := s.sessionRepo.InvalidateUserSessions(ctx, user.ID, s.cfg.JWT.RefreshExpiry); err != nil {
		s.logger.Warn("Không thể invalidate sessions sau force change password", zap.Error(err))
	}

	s.logger.Info("Người dùng đã force đổi mật khẩu thành công", zap.Uint64("user_id", user.ID))
	return nil
}

type OTPService struct {
	otpRepo     *repository.OTPRepository
	mailService *MailService
	logger      *zap.Logger
	pepper      string
	maxAttempts int
}

func NewOTPService(otpRepo *repository.OTPRepository, mailService *MailService, logger *zap.Logger, cfg *config.Config) *OTPService {
	return &OTPService{
		otpRepo:     otpRepo,
		mailService: mailService,
		logger:      logger,
		pepper:      cfg.Security.OTPPepper,
		maxAttempts: cfg.Security.OTPMaxAttempts,
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

	codeHash := hash.HashOTP(otp, s.pepper)
	if err := s.otpRepo.Create(ctx, userID, codeHash, otpType, expiresAt); err != nil {
		return nil, fmt.Errorf("otpRepo.Create: %w", err)
	}

	sendFunc := func() {
		if err := s.mailService.SendVerificationEmail(email, otp); err != nil {
			s.logger.Error("Không thể gửi email OTP", zap.String("email", email), zap.Error(err))
			// Local/dev: mail fail vẫn cho test tiếp (Docker SMTP port hay gãy trên Windows).
			s.logger.Warn("FALLBACK OTP (dev) — chỉ dùng khi SMTP lỗi",
				zap.String("email", email),
				zap.String("otp", otp),
			)
		}
	}

	return sendFunc, nil
}

// VerifyOTP kiểm tra mã OTP đầu vào (so sánh hash constant-time).
// Nếu gọi trong transaction, truyền txCtx để FOR UPDATE đúng connection.
func (s *OTPService) VerifyOTP(ctx context.Context, userID uint64, otpType string, inputCode string) error {
	otpCode, err := s.otpRepo.GetLatestValidCodeForUpdate(ctx, userID, otpType)
	if err != nil {
		return fmt.Errorf("otpRepo.GetLatestValidCodeForUpdate: %w", err)
	}
	if otpCode == nil {
		return apperror.ErrOTPExpired
	}

	if otpCode.Attempts >= s.maxAttempts {
		return apperror.ErrOTPMaxAttempts
	}

	inputHash := hash.HashOTP(inputCode, s.pepper)
	if subtle.ConstantTimeCompare([]byte(otpCode.Code), []byte(inputHash)) != 1 {
		attempts, err := s.otpRepo.IncrementAttempts(ctx, otpCode.ID)
		if err != nil {
			return fmt.Errorf("otpRepo.IncrementAttempts: %w", err)
		}
		if attempts >= s.maxAttempts {
			return apperror.ErrOTPMaxAttempts
		}
		return apperror.ErrOTPInvalid
	}

	if err := s.otpRepo.MarkAsUsed(ctx, otpCode.ID); err != nil {
		return fmt.Errorf("otpRepo.MarkAsUsed: %w", err)
	}

	return nil
}
