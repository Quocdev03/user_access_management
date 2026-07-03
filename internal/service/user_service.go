package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
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

func NewUserService(
	userRepo *repository.UserRepository,
	otpService *OTPService,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	mailService *MailService,
	txManager *database.TxManager,
	cfg *config.Config,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userRepo:    userRepo,
		otpService:  otpService,
		roleRepo:    roleRepo,
		sessionRepo: sessionRepo,
		mailService: mailService,
		txManager:   txManager,
		cfg:         cfg,
		logger:      logger,
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
	return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		if req.FullName != nil {
			userForUpdate.FullName = *req.FullName
		}
		if req.Phone != nil {
			userForUpdate.Phone = *req.Phone
		}
		if req.DateOfBirth != nil {
			parsedDate, err := time.Parse("2006-01-02", *req.DateOfBirth)
			if err != nil {
				return apperror.ErrBadRequest.WithMessage("Định dạng ngày sinh không hợp lệ (YYYY-MM-DD)")
			}
			userForUpdate.DateOfBirth = parsedDate
		}
		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})
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

	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "email_change", user.Email, 3, 1*time.Minute)
	if isLimited {
		return apperror.ErrRateLimited
	}

	existing, err := s.userRepo.FindByEmail(ctx, req.NewEmail)
	if err != nil {
		return fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	if existing != nil {
		return apperror.ErrConflict
	}

	var sendEmailOld, sendEmailNew func()
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		fnOld, err := s.otpService.CreateAndSendOTP(txCtx, userID, user.Email, constant.OTPTypeChangeEmailOld, time.Now().UTC().Add(15*time.Minute))
		if err != nil {
			return err
		}

		fnNew, err := s.otpService.CreateAndSendOTP(txCtx, userID, req.NewEmail, constant.OTPTypeChangeEmailNew, time.Now().UTC().Add(15*time.Minute))
		if err != nil {
			return err
		}

		if err := s.sessionRepo.SetEmailChangePending(txCtx, userID, req.NewEmail, 15*time.Minute); err != nil {
			return fmt.Errorf("sessionRepo.SetEmailChangePending: %w", err)
		}
		
		sendEmailOld = fnOld
		sendEmailNew = fnNew
		return nil
	})

	if err != nil {
		return err
	}

	go sendEmailOld()
	go sendEmailNew()
	return nil
}

func (s *UserService) VerifyEmailChange(ctx context.Context, userID uint64, req dto.VerifyEmailChangeRequest) error {
	newEmail, err := s.sessionRepo.GetEmailChangePending(ctx, userID)
	if err != nil {
		return fmt.Errorf("sessionRepo.GetEmailChangePending: %w", err)
	}
	if newEmail == "" {
		return apperror.ErrBadRequest.WithMessage("Không có yêu cầu đổi email nào đang chờ xử lý hoặc đã hết hạn")
	}

	var oldEmail string
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Verify both OTPs
		if err := s.otpService.VerifyOTP(txCtx, userID, constant.OTPTypeChangeEmailOld, req.OldOTP); err != nil {
			return err
		}
		if err := s.otpService.VerifyOTP(txCtx, userID, constant.OTPTypeChangeEmailNew, req.NewOTP); err != nil {
			return err
		}

		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		oldEmail = userForUpdate.Email
		userForUpdate.Email = newEmail
		userForUpdate.EmailVerified = true
		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
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
	var oldAvatarURL string

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		if userForUpdate.AvatarURL != nil {
			oldAvatarURL = *userForUpdate.AvatarURL
		}

		userForUpdate.AvatarURL = &avatarURL
		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if oldAvatarURL != "" {
		if strings.HasPrefix(oldAvatarURL, "/uploads/avatars/") {
			oldPath := filepath.Join(".", strings.TrimPrefix(oldAvatarURL, "/"))
			if err := os.Remove(filepath.Clean(oldPath)); err != nil {
				s.logger.Warn("Không thể xóa avatar cũ", zap.String("path", oldPath), zap.Error(err))
			}
		}
	}

	return &dto.UploadAvatarResponse{
		AvatarURL: avatarURL,
	}, nil
}

func (s *UserService) DeleteAvatar(ctx context.Context, userID uint64) error {
	var oldAvatarURL string
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		userForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("userRepo.FindByIDForUpdate: %w", err)
		}
		if userForUpdate == nil {
			return apperror.ErrNotFound
		}

		if userForUpdate.AvatarURL == nil || *userForUpdate.AvatarURL == "" {
			return nil
		}

		oldAvatarURL = *userForUpdate.AvatarURL
		userForUpdate.AvatarURL = nil
		if err := s.userRepo.UpdateUser(txCtx, userForUpdate); err != nil {
			return fmt.Errorf("userRepo.UpdateUser: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if oldAvatarURL != "" {
		if strings.HasPrefix(oldAvatarURL, "/uploads/avatars/") {
			oldPath := filepath.Join(".", strings.TrimPrefix(oldAvatarURL, "/"))
			if err := os.Remove(filepath.Clean(oldPath)); err != nil {
				s.logger.Warn("Không thể xóa avatar cũ", zap.String("path", oldPath), zap.Error(err))
			}
		}
	}

	return nil
}

func (s *UserService) ResendChangeEmailOTP(ctx context.Context, userID uint64) error {
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
	if newEmail == "" {
		return apperror.ErrBadRequest.WithMessage("Không có yêu cầu đổi email nào đang chờ xử lý")
	}

	isLimited, _ := s.sessionRepo.IncrementRateLimit(ctx, "resend_email_otp", user.Email, 3, 1*time.Minute)
	if isLimited {
		return apperror.ErrRateLimited
	}

	var sendEmailOld, sendEmailNew func()
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		fnOld, err := s.otpService.CreateAndSendOTP(txCtx, userID, user.Email, constant.OTPTypeChangeEmailOld, time.Now().UTC().Add(15*time.Minute))
		if err != nil {
			return err
		}

		fnNew, err := s.otpService.CreateAndSendOTP(txCtx, userID, newEmail, constant.OTPTypeChangeEmailNew, time.Now().UTC().Add(15*time.Minute))
		if err != nil {
			return err
		}

		sendEmailOld = fnOld
		sendEmailNew = fnNew
		return nil
	})

	if err != nil {
		return err
	}

	go sendEmailOld()
	go sendEmailNew()
	return nil
}
