package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

type UserService struct {
	userRepo    *repository.UserRepository
	otpService  *OTPService
	roleRepo    *repository.RoleRepository
	sessionRepo *repository.SessionRepository
	mailService *MailService
	txManager   *database.TxManager
	cfg         *config.Config
	logger      *zap.Logger
}

type UserServiceParams struct {
	UserRepo    *repository.UserRepository
	OtpService  *OTPService
	RoleRepo    *repository.RoleRepository
	SessionRepo *repository.SessionRepository
	MailService *MailService
	TxManager   *database.TxManager
	Cfg         *config.Config
	Logger      *zap.Logger
}

func NewUserService(params UserServiceParams) *UserService {
	return &UserService{
		userRepo:    params.UserRepo,
		otpService:  params.OtpService,
		roleRepo:    params.RoleRepo,
		sessionRepo: params.SessionRepo,
		mailService: params.MailService,
		txManager:   params.TxManager,
		cfg:         params.Cfg,
		logger:      params.Logger,
	}
}

func (s *UserService) GetProfile(ctx context.Context, userID uint64) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return nil, apperror.ErrNotFound
	}

	roles, err := s.roleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		roles = []string{constant.RoleUser}
	}

	return &dto.UserProfileResponse{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		FullName:      user.FullName,
		Phone:         user.Phone,
		AvatarURL:     user.AvatarURL,
		Status:        string(user.Status),
		DateOfBirth:   user.DateOfBirth.Format("2006-01-02"),
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt,
		Roles:         roles,
	}, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uint64, req dto.UpdateProfileRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	user.FullName = req.FullName
	user.Phone = req.Phone

	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("userRepo.UpdateUser: %w", err)
	}
	return nil
}

func (s *UserService) RequestEmailChange(ctx context.Context, userID uint64, req dto.RequestEmailChangeRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	if !hash.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		return apperror.ErrInvalidCredentials
	}

	existing, err := s.userRepo.FindByEmail(ctx, req.NewEmail)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if existing != nil {
		return apperror.ErrConflict
	}

	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "email_change", user.Email, 3, 1*time.Minute)
	if isLimited {
		return apperror.ErrRateLimited
	}

	sendEmail, err := s.otpService.CreateAndSendOTP(ctx, userID, user.Email, constant.OTPTypeChangeEmail, time.Now().UTC().Add(5*time.Minute))
	if err != nil {
		return err
	}

	if err := s.sessionRepo.SetEmailChangePending(ctx, userID, req.NewEmail, 15*time.Minute); err != nil {
		return fmt.Errorf("sessionRepo.SetEmailChangePending: %w", err)
	}
	
	go sendEmail()

	return nil
}

func (s *UserService) VerifyOldEmail(ctx context.Context, userID uint64, req dto.VerifyOldEmailRequest) (*dto.VerifyOldEmailResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return nil, apperror.ErrNotFound
	}

	newEmail, err := s.sessionRepo.GetEmailChangePending(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.GetEmailChangePending: %w", err)
	}
	if newEmail == "" {
		return nil, apperror.ErrBadRequest.WithMessage("Không có yêu cầu đổi email nào đang chờ xử lý")
	}

	var changeToken string
	var sendEmail func()
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.otpService.VerifyOTP(txCtx, userID, constant.OTPTypeChangeEmail, req.OTP); err != nil {
			return err
		}

		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			return fmt.Errorf("failed to generate random token: %w", err)
		}
		changeToken = hex.EncodeToString(tokenBytes)

		fn, err := s.otpService.CreateAndSendOTP(txCtx, userID, newEmail, constant.OTPTypeChangeEmail, time.Now().UTC().Add(5*time.Minute))
		if err != nil {
			return err
		}
		sendEmail = fn

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := s.sessionRepo.SetEmailChangeToken(ctx, userID, changeToken, 15*time.Minute); err != nil {
		return nil, fmt.Errorf("sessionRepo.SetEmailChangeToken: %w", err)
	}
	
	go sendEmail()

	return &dto.VerifyOldEmailResponse{
		EmailChangeToken: changeToken,
	}, nil
}

func (s *UserService) VerifyNewEmail(ctx context.Context, userID uint64, req dto.VerifyNewEmailRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	newEmail, err := s.sessionRepo.GetEmailChangePending(ctx, userID)
	if err != nil {
		return fmt.Errorf("sessionRepo.GetEmailChangePending: %w", err)
	}
	storedToken, err := s.sessionRepo.GetEmailChangeToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("sessionRepo.GetEmailChangeToken: %w", err)
	}

	if newEmail == "" || storedToken == "" || subtle.ConstantTimeCompare([]byte(storedToken), []byte(req.EmailChangeToken)) != 1 {
		return apperror.ErrBadRequest.WithMessage("Phiên giao dịch đổi email không hợp lệ hoặc đã hết hạn")
	}

	oldEmail := user.Email

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.otpService.VerifyOTP(txCtx, userID, constant.OTPTypeChangeEmail, req.OTP); err != nil {
			return err
		}

		user.Email = newEmail
		user.EmailVerified = true
		if err := s.userRepo.UpdateUser(txCtx, user); err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				return apperror.ErrConflict
			}
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}

		if err := s.sessionRepo.DeleteByUserID(txCtx, userID); err != nil {
			return fmt.Errorf("sessionRepo.DeleteByUserID: %w", err)
		}
		_ = s.sessionRepo.RevokeAllUserTokens(txCtx, userID, s.cfg.JWT.RefreshExpiry)

		return nil
	})

	if err != nil {
		return err
	}

	_ = s.sessionRepo.DeleteEmailChangePending(ctx, userID)

	go func() {
		_ = s.mailService.SendEmailChangeNotification(oldEmail, newEmail)
	}()

	return nil
}

func (s *UserService) UploadAvatar(ctx context.Context, userID uint64, file *multipart.FileHeader) (*dto.UploadAvatarResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return nil, apperror.ErrNotFound
	}

	if file.Size > 2*1024*1024 {
		return nil, apperror.ErrBadRequest.WithMessage("Kích thước tệp tin không được vượt quá 2MB")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, apperror.ErrBadRequest.WithMessage("Định dạng tệp không được hỗ trợ. Chỉ cho phép JPEG, PNG, WebP")
	}

	uploadDir := "./uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("không thể tạo thư mục upload: %w", err)
	}

	fileName := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open upload file: %w", err)
	}
	defer src.Close()

	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file for content type detection: %w", err)
	}
	contentType := http.DetectContentType(buffer[:n])
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return nil, apperror.ErrBadRequest.WithMessage("Nội dung tệp không hợp lệ. Chỉ cho phép JPEG, PNG, WebP")
	}
	
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			s.logger.Error("Lỗi khi đóng file avatar sau ghi", zap.Error(closeErr))
		}
	}()

	if _, err = io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	avatarURL := fmt.Sprintf("/uploads/avatars/%s", fileName)

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		if strings.HasPrefix(*user.AvatarURL, "/uploads/avatars/") {
			oldPath := filepath.Join(".", *user.AvatarURL)
			_ = os.Remove(filepath.Clean(oldPath))
		}
	}

	user.AvatarURL = &avatarURL
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("userRepo.UpdateUser: %w", err)
	}

	return &dto.UploadAvatarResponse{
		AvatarURL: avatarURL,
	}, nil
}

func (s *UserService) DeleteAvatar(ctx context.Context, userID uint64) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		return nil
	}

	if strings.HasPrefix(*user.AvatarURL, "/uploads/avatars/") {
		oldPath := filepath.Join(".", *user.AvatarURL)
		_ = os.Remove(filepath.Clean(oldPath))
	}

	user.AvatarURL = nil
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("userRepo.UpdateUser: %w", err)
	}
	return nil
}
