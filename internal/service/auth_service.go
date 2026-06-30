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

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)
	VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error
	ResendVerificationEmail(ctx context.Context, req dto.ResendVerificationEmailRequest) error
	Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error)
	Refresh(ctx context.Context, req dto.RefreshTokenRequest, ipAddress, userAgent string) (*dto.RefreshTokenResponse, error)
	Logout(ctx context.Context, rawToken string, claims *jwt.Claims) error
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
	ChangePassword(ctx context.Context, userID uint64, req dto.ChangePasswordRequest) error
}

type authService struct {
	userRepo          repository.UserRepository
	otpRepo           repository.OTPRepository
	roleRepo          repository.RoleRepository
	sessionRepo       repository.SessionRepository
	passwordResetRepo repository.PasswordResetRepository
	mailService       MailService
	cfg               *config.Config
	logger            *zap.Logger
}

func NewAuthService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	roleRepo repository.RoleRepository,
	sessionRepo repository.SessionRepository,
	passwordResetRepo repository.PasswordResetRepository,
	mailService MailService,
	cfg *config.Config,
	logger *zap.Logger,
) AuthService {
	return &authService{
		userRepo:          userRepo,
		otpRepo:           otpRepo,
		roleRepo:          roleRepo,
		sessionRepo:       sessionRepo,
		passwordResetRepo: passwordResetRepo,
		mailService:       mailService,
		cfg:               cfg,
		logger:            logger,
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
func (s *authService) generateTokenPair(userID uint64) (string, string, error) {
	accessToken, _, err := jwt.GenerateToken(userID, "access", s.cfg.JWT.AccessExpiry, s.cfg.JWT.Secret)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err := jwt.GenerateToken(userID, "refresh", s.cfg.JWT.RefreshExpiry, s.cfg.JWT.Secret)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {

	// Hash password
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash.HashPassword: %w", err)
	}

	// Parse DateOfBirth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, apperror.NewAppError("ERR_INVALID_DATE_FORMAT", "Ngày sinh không đúng định dạng YYYY-MM-DD", 400)
	}

	// Tạo User (trạng thái inactive)
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

	if err := s.userRepo.Create(ctx, user); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, apperror.ErrConflict
		}
		return nil, fmt.Errorf("userRepo.Create: %w", err)
	}

	// Gán role 'user'
	role, err := s.roleRepo.FindByName(ctx, "user")
	if err != nil || role == nil {
		s.logger.Warn("Không tìm được default role", zap.Error(err))
	} else if err := s.roleRepo.AssignRoleToUser(ctx, user.ID, role.ID); err != nil {
		s.logger.Warn("Không gán được role cho user", zap.Uint64("user_id", user.ID), zap.Error(err))
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

	// Gửi email OTP thật sự (UC-25)
	if err := s.mailService.SendVerificationEmail(user.Email, otp); err != nil {
		s.logger.Error("Không thể gửi email OTP đăng ký", zap.String("email", user.Email), zap.Error(err))
		// Có thể trả lỗi hoặc bỏ qua lỗi gửi email tùy yêu cầu dự án. Ở đây ta log lỗi nhưng vẫn cho phép đăng ký thành công.
	}

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

	if subtle.ConstantTimeCompare([]byte(otpCode.Code), []byte(req.OTP)) != 1 {
		attempts, _ := s.otpRepo.IncrementAttempts(ctx, otpCode.ID)
		if attempts >= maxOTPAttempts {
			return apperror.NewAppError("OTP_MAX_ATTEMPTS", "Bạn đã nhập sai OTP quá 5 lần, vui lòng yêu cầu mã mới", 400)
		}
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

func (s *authService) ResendVerificationEmail(ctx context.Context, req dto.ResendVerificationEmailRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		// Để tránh dò tìm email (enumeration), có thể trả về lỗi chung chung, nhưng vì mục đích UX tốt ta sẽ báo rõ:
		return apperror.NewAppError("ERR_EMAIL_NOT_FOUND", "Email chưa được đăng ký", 400)
	}

	if user.EmailVerified {
		return apperror.NewAppError("ALREADY_VERIFIED", "Tài khoản đã được xác thực, vui lòng đăng nhập", 400)
	}

	// Sinh OTP mới
	otp, err := generateOTP()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(otpExpiry)

	// Ghi nhận OTP vào bảng otp_codes
	if err := s.otpRepo.Create(ctx, user.ID, otp, "email_verification", expiresAt); err != nil {
		return fmt.Errorf("otpRepo.Create: %w", err)
	}

	// Gửi lại email
	if err := s.mailService.SendVerificationEmail(user.Email, otp); err != nil {
		s.logger.Error("Không thể gửi email OTP (Resend)", zap.String("email", user.Email), zap.Error(err))
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.Info("Đã gửi lại email xác thực", zap.String("email", user.Email))
	return nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if user == nil {
		// VULN-003: Chống dò tìm email (Timing Attack) bằng cách băm mật khẩu với dummy hash
		_ = hash.CheckPassword(req.Password, dummyBcryptHash)
		return nil, apperror.ErrInvalidCredentials
	}

	// Kiểm tra trạng thái locked (với tự động unlock nếu hết hạn)
	if user.Status == model.StatusLocked {
		if user.LockedUntil == nil || user.LockedUntil.After(time.Now()) {
			return nil, apperror.ErrAccountLocked
		}
		// Hết thời gian khóa -> cập nhật struct tạm (DB sẽ tự update ở UpdateLastLogin hoặc khóa lại ở handleFailedLogin)
		user.Status = model.StatusActive
		user.FailedLoginAttempts = 0
	}

	if user.Status == model.StatusInactive {
		return nil, apperror.NewAppError("INACTIVE_ACCOUNT", "Vui lòng xác thực email trước khi đăng nhập", 403)
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		return nil, s.handleFailedLogin(ctx, user)
	}

	// Đăng nhập thành công -> update last_login và reset counter
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	s.logger.Info("Đăng nhập thành công", zap.String("email", user.Email), zap.Uint64("user_id", user.ID))

	// Sinh JWT Access Token & Refresh Token
	accessToken, refreshToken, err := s.generateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	// Lưu session vào MySQL
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
		User: dto.UserInfoResponse{ // B3: Dùng named struct thay vì anonymous
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

func (s *authService) Refresh(ctx context.Context, req dto.RefreshTokenRequest, ipAddress, userAgent string) (*dto.RefreshTokenResponse, error) {
	// Parse và validate refresh token
	claims, err := jwt.ParseToken(req.RefreshToken, s.cfg.JWT.Secret)
	if err != nil {
		return nil, apperror.NewAppError("ERR_REFRESH_INVALID", "Mã refresh token không hợp lệ hoặc đã hết hạn", 401)
	}

	if claims.Type != "refresh" {
		return nil, apperror.NewAppError("ERR_REFRESH_INVALID", "Mã token không phải refresh token", 401)
	}

	// Tìm session tương ứng bằng hash của refresh token
	refreshHash := hashToken(req.RefreshToken)

	// VULN-002: Phát hiện Token Reuse (Sử dụng lại Refresh Token đã xoay vòng)
	isRevoked, _ := s.sessionRepo.IsRefreshTokenRevoked(ctx, refreshHash)
	if isRevoked {
		s.logger.Warn("Phát hiện Token Reuse (Refresh Token bị đánh cắp)", zap.Uint64("user_id", claims.UserID))
		// Xóa toàn bộ phiên đăng nhập của người dùng để bảo vệ tài khoản
		_ = s.sessionRepo.DeleteByUserID(ctx, claims.UserID)
		return nil, apperror.NewAppError("ERR_TOKEN_REUSE", "Phát hiện truy cập bất thường, vui lòng đăng nhập lại", 401)
	}

	session, err := s.sessionRepo.FindByRefreshTokenHash(ctx, refreshHash)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.FindByRefreshTokenHash: %w", err)
	}

	if session == nil {
		return nil, apperror.NewAppError("ERR_REFRESH_INVALID", "Không tìm thấy session hoặc refresh token đã bị thu hồi", 401)
	}

	// Kiểm tra hết hạn trong DB (phòng hờ)
	if session.ExpiresAt.Before(time.Now()) {
		_ = s.sessionRepo.DeleteByRefreshTokenHash(ctx, refreshHash)
		return nil, apperror.NewAppError("ERR_REFRESH_INVALID", "Refresh token đã hết hạn", 401)
	}

	// Kiểm tra trạng thái user hiện tại
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil || user == nil || user.Status != model.StatusActive {
		_ = s.sessionRepo.DeleteByUserID(ctx, claims.UserID)
		return nil, apperror.NewAppError("ERR_ACCOUNT_DISABLED", "Tài khoản của bạn đã bị khóa hoặc vô hiệu hóa", 403)
	}

	// Xoay vòng token (Rotation): Tạo mới cặp access và refresh token
	newAccessToken, newRefreshToken, err := s.generateTokenPair(claims.UserID)
	if err != nil {
		return nil, err
	}

	// UPDATE session cũ thay vì Delete + Create
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

	// VULN-002: Thêm refresh token cũ vào danh sách thu hồi (blacklist Redis) với TTL = thời gian sống còn lại
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		_ = s.sessionRepo.AddRevokedRefreshToken(ctx, refreshHash, ttl)
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *authService) Logout(ctx context.Context, rawToken string, claims *jwt.Claims) error {
	// Tính TTL còn lại của access token để blacklist trong Redis
	expTime := claims.ExpiresAt.Time
	ttl := time.Until(expTime)

	if ttl > 0 {
		// Thêm JTI của access token vào Redis blacklist
		if err := s.sessionRepo.AddToBlacklist(ctx, claims.ID, ttl); err != nil {
			s.logger.Warn("Không thể blacklist access token", zap.String("jti", claims.ID), zap.Error(err))
		}
	}

	// Xóa session trong MySQL bằng token_hash của access token hiện tại
	tokenHash := hashToken(rawToken)
	if err := s.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("sessionRepo.DeleteByTokenHash: %w", err)
	}

	s.logger.Info("Đăng xuất thành công", zap.Uint64("user_id", claims.UserID))
	return nil
}

func (s *authService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}

	// UC-07: Luôn trả thành công dù email không tồn tại để chống dò email
	if user == nil {
		s.logger.Info("Yêu cầu quên mật khẩu cho email không tồn tại", zap.String("email", req.Email))
		return nil
	}

	// Kiểm tra Rate Limit để chống spam
	isLimited, _ := s.sessionRepo.IsRateLimited(ctx, "forgot_pw", req.Email)
	if isLimited {
		return apperror.NewAppError("ERR_RATE_LIMIT", "Bạn thao tác quá nhanh, vui lòng chờ 1 phút trước khi yêu cầu lại", 429)
	}
	_ = s.sessionRepo.SetRateLimit(ctx, "forgot_pw", req.Email, 1*time.Minute)

	// Sinh token ngẫu nhiên
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(1 * time.Hour) // Token hết hạn sau 1 giờ

	// Vô hiệu hóa các token khôi phục mật khẩu cũ của user
	_ = s.passwordResetRepo.InvalidateAllUserTokens(ctx, user.ID)

	// Lưu token mới vào DB
	if err := s.passwordResetRepo.Create(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("passwordResetRepo.Create: %w", err)
	}

	// Gửi email chứa link khôi phục
	if err := s.mailService.SendPasswordResetEmail(user.Email, token); err != nil {
		s.logger.Error("Lỗi khi gửi email khôi phục mật khẩu", zap.Error(err))
		return nil // LUÔN trả về nil để tránh lộ thông tin tồn tại email
	}

	s.logger.Info("Đã gửi email khôi phục mật khẩu", zap.String("email", user.Email))
	return nil
}

func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	tokenHash := hashToken(req.Token)
	resetToken, err := s.passwordResetRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("passwordResetRepo.FindByTokenHash: %w", err)
	}

	if resetToken == nil || resetToken.IsUsed || resetToken.ExpiresAt.Before(time.Now()) {
		return apperror.NewAppError("ERR_INVALID_TOKEN", "Link khôi phục mật khẩu không hợp lệ hoặc đã hết hạn", 400)
	}

	// Validate và Hash mật khẩu mới
	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	// Đánh dấu token đã sử dụng BẰNG ATOMIC QUERY để chặn race condition
	if err := s.passwordResetRepo.MarkAsUsed(ctx, resetToken.ID); err != nil {
		return apperror.NewAppError("ERR_INVALID_TOKEN", "Link khôi phục mật khẩu đã được sử dụng hoặc hết hạn", 400)
	}

	// Dọn dẹp các token cũ khác của user
	_ = s.passwordResetRepo.InvalidateAllUserTokens(ctx, resetToken.UserID)

	// Vô hiệu hóa tất cả các session của user (ép đăng nhập lại) TRƯỚC khi update DB
	if err := s.sessionRepo.DeleteByUserID(ctx, resetToken.UserID); err != nil {
		s.logger.Error("Không thể thu hồi session khi reset password", zap.Error(err))
		return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
	}

	// Cập nhật mật khẩu
	if err := s.userRepo.UpdatePassword(ctx, resetToken.UserID, hashedPassword); err != nil {
		return fmt.Errorf("userRepo.UpdatePassword: %w", err)
	}

	// Mở khóa tài khoản nếu đang bị khóa
	if err := s.userRepo.UnlockAccount(ctx, resetToken.UserID); err != nil {
		s.logger.Error("Không thể mở khóa tài khoản sau khi reset password", zap.Error(err))
	}

	s.logger.Info("Người dùng đã khôi phục mật khẩu thành công", zap.Uint64("user_id", resetToken.UserID))
	return nil
}

func (s *authService) ChangePassword(ctx context.Context, userID uint64, req dto.ChangePasswordRequest) error {
	// Lấy thông tin user để verify mật khẩu cũ
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	if user.Status == model.StatusLocked {
		return apperror.NewAppError("ACCOUNT_LOCKED", "Tài khoản đang bị khóa, không thể đổi mật khẩu", 423)
	}

	if !hash.CheckPassword(req.OldPassword, user.PasswordHash) {
		return s.handleFailedLogin(ctx, user)
	}

	if req.OldPassword == req.NewPassword {
		return apperror.NewAppError("ERR_SAME_PASSWORD", "Mật khẩu mới không được trùng với mật khẩu cũ", 400)
	}

	// Hash mật khẩu mới
	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash.HashPassword: %w", err)
	}

	// Thu hồi tất cả session (đăng xuất khỏi mọi thiết bị) TRƯỚC khi đổi pass
	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		s.logger.Error("Không thể thu hồi session khi đổi password", zap.Error(err))
		return fmt.Errorf("không thể thu hồi phiên đăng nhập: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return fmt.Errorf("userRepo.UpdatePassword: %w", err)
	}

	s.logger.Info("Người dùng đã đổi mật khẩu thành công", zap.Uint64("user_id", userID))
	return nil
}

// handleFailedLogin xử lý logic tăng số lần sai và khóa tài khoản
func (s *authService) handleFailedLogin(ctx context.Context, user *model.User) error {
	attempts, _ := s.userRepo.IncrementFailedLogins(ctx, user.ID)
	
	if attempts >= maxFailedAttempts {
		lockedUntil := time.Now().Add(lockDuration)
		_ = s.userRepo.LockAccount(ctx, user.ID, lockedUntil)
		s.logger.Warn("Tài khoản bị khóa do sai mật khẩu nhiều lần",
			zap.String("email", user.Email),
			zap.Int("failed_attempts", attempts),
		)
		return apperror.NewAppError("ACCOUNT_LOCKED",
			fmt.Sprintf("Tài khoản bị khóa %d phút do nhập sai mật khẩu %d lần",
				int(lockDuration.Minutes()), maxFailedAttempts),
			423,
		)
	}
	return apperror.ErrInvalidCredentials
}
