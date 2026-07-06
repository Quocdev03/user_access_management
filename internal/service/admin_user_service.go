package service

import (
	"context"
	"fmt"
	"time"

	"github.com/quocdev03/user-access-management/internal/constant"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/quocdev03/user-access-management/pkg/hash"
	"go.uber.org/zap"
)

type AdminUserService struct {
	userRepo     *repository.UserRepository
	roleRepo     *repository.RoleRepository
	sessionRepo  *repository.SessionRepository
	auditLogRepo *repository.AuditLogRepository
	mailService  *MailService
	txManager    *database.TxManager
	logger       *zap.Logger
}

func NewAdminUserService(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	auditLogRepo *repository.AuditLogRepository,
	mailService *MailService,
	txManager *database.TxManager,
	logger *zap.Logger,
) *AdminUserService {
	return &AdminUserService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		sessionRepo:  sessionRepo,
		auditLogRepo: auditLogRepo,
		mailService:  mailService,
		txManager:    txManager,
		logger:       logger,
	}
}

func (s *AdminUserService) logAudit(ctx context.Context, adminID uint64, action, resourceID, ip, ua, status string) {
	resource := "users"
	_ = s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:     &adminID,
		Action:     action,
		Resource:   &resource,
		ResourceID: &resourceID,
		IPAddress:  &ip,
		UserAgent:  &ua,
		Status:     status,
	})
}

func (s *AdminUserService) ListUsers(ctx context.Context, req dto.AdminListUsersRequest) (*dto.AdminListUsersResponse, error) {
	if req.Page <= 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	offset := (req.Page - 1) * req.PerPage

	users, total, err := s.userRepo.ListUsers(
		ctx,
		req.Username,
		req.Email,
		req.Status,
		req.Role,
		req.Page,
		offset,
		req.SortBy,
		req.SortOrder,
	)

	if err != nil {
		return nil, fmt.Errorf("userRepo.ListUsers: %w", err)
	}

	userIDs := make([]uint64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	rolesMap, err := s.roleRepo.GetRolesByUserIDs(ctx, userIDs)
	if err != nil {
		s.logger.Warn("Lỗi khi fetch roles", zap.Error(err))
	}

	list := make([]dto.AdminUserListItem, 0, len(users))
	for _, u := range users {
		roles := rolesMap[u.ID]
		if len(roles) == 0 {
			roles = []string{constant.RoleUser}
		}

		list = append(list, dto.AdminUserListItem{
			ID:            u.ID,
			Username:      u.Username,
			Email:         u.Email,
			FullName:      u.FullName,
			Phone:         u.Phone,
			Status:        string(u.Status),
			EmailVerified: u.EmailVerified,
			DateOfBirth:   u.DateOfBirth.Format("2006-01-02"),
			AvatarURL:     u.AvatarURL,
			Roles:         roles,
			CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		})
	}

	totalPages := (int(total) + req.PerPage - 1) / req.PerPage

	return &dto.AdminListUsersResponse{
		Users: list,
		Meta: model.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *AdminUserService) GetUserDetail(ctx context.Context, userID uint64) (*dto.AdminUserListItem, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}

	if user == nil {
		return nil, apperror.ErrNotFound
	}

	roles, _ := s.roleRepo.GetRolesByUserID(ctx, userID)
	if len(roles) == 0 {
		roles = []string{constant.RoleUser}
	}

	return &dto.AdminUserListItem{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		FullName:      user.FullName,
		Phone:         user.Phone,
		Status:        string(user.Status),
		EmailVerified: user.EmailVerified,
		DateOfBirth:   user.DateOfBirth.Format("2006-01-02"),
		AvatarURL:     user.AvatarURL,
		Roles:         roles,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *AdminUserService) UpdateUser(ctx context.Context, adminID, targetID uint64, req dto.AdminUpdateUserRequest, ip, ua string) error {
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		user, err := s.userRepo.FindByIDForUpdate(txCtx, targetID)
		if err != nil {
			return err
		}
		if user == nil {
			return apperror.ErrNotFound
		}

		var needsRevokeSession bool

		if req.FullName != nil {
			user.FullName = *req.FullName
		}
		if req.Phone != nil {
			user.Phone = *req.Phone
		}
		if req.Email != nil && *req.Email != user.Email {
			// Kiểm tra trùng Email
			existingUser, _ := s.userRepo.FindByEmail(txCtx, *req.Email)
			if existingUser != nil && existingUser.ID != user.ID {
				return apperror.ErrConflict.WithMessage("Email đã được sử dụng bởi người dùng khác")
			}
			user.Email = *req.Email
			// Nếu đổi Email mà Admin không explicitly set verified = true, mặc định là unverified
			if req.EmailVerified == nil || !*req.EmailVerified {
				user.EmailVerified = false
				user.Status = constant.StatusInactive
				needsRevokeSession = true
			}
		}
		if req.EmailVerified != nil {
			user.EmailVerified = *req.EmailVerified
			if !*req.EmailVerified {
				user.Status = constant.StatusInactive
				needsRevokeSession = true
			} else if user.Status == constant.StatusInactive {
				// Nếu bật EmailVerified thành true, tự động mở khóa trạng thái Inactive
				user.Status = constant.StatusActive
			}
		}
		if req.DateOfBirth != nil {
			dob, err := time.Parse("2006-01-02", *req.DateOfBirth)
			if err != nil {
				return apperror.ErrInvalidDateFormat
			}
			user.DateOfBirth = dob
		}
		if req.AvatarURL != nil {
			user.AvatarURL = req.AvatarURL
		}

		if err := s.userRepo.UpdateUser(txCtx, user); err != nil {
			return err
		}

		// Nếu tài khoản bị set về Inactive, revoke sessions để ép văng ra
		if needsRevokeSession {
			_ = s.sessionRepo.RevokeAllUserTokens(txCtx, user.ID, 7*24*time.Hour) // Dùng mặc định 7 ngày do AdminUserService không chứa config
		}

		return nil
	})

	if err != nil {
		s.logAudit(ctx, adminID, "ADMIN_UPDATE_USER", fmt.Sprint(targetID), ip, ua, "failure")
		return err
	}

	s.logAudit(ctx, adminID, "ADMIN_UPDATE_USER", fmt.Sprint(targetID), ip, ua, "success")
	return nil
}

func (s *AdminUserService) ChangeUserStatus(ctx context.Context, adminID, targetID uint64, req dto.AdminChangeStatusRequest, ip, ua string) error {
	if adminID == targetID {
		return apperror.ErrBadRequest.WithMessage("Không thể tự khoá tài khoản của chính mình")
	}

	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		user, err := s.userRepo.FindByIDForUpdate(txCtx, targetID)
		if err != nil {
			return err
		}
		if user == nil {
			return apperror.ErrNotFound
		}

		user.Status = constant.UserStatus(req.Status)
		if user.Status == constant.StatusLocked {
			if err := s.sessionRepo.DeleteByUserID(txCtx, targetID); err != nil {
				s.logger.Warn("không thể xoá session khi lock user", zap.Error(err), zap.Uint64("userID", targetID))
			}
		}
		return s.userRepo.UpdateUser(txCtx, user)
	})

	if err != nil {
		s.logAudit(ctx, adminID, "ADMIN_CHANGE_STATUS", fmt.Sprint(targetID), ip, ua, "failure")
		return err
	}

	if req.Status == string(constant.StatusLocked) {
		if err := s.sessionRepo.RevokeAllUserTokens(ctx, targetID, 7*24*time.Hour); err != nil {
			s.logger.Warn("không thể revoke token redis khi lock user", zap.Error(err), zap.Uint64("userID", targetID))
		}
	}

	s.logAudit(ctx, adminID, "ADMIN_CHANGE_STATUS", fmt.Sprint(targetID), ip, ua, "success")
	return nil
}

func (s *AdminUserService) ResetUserPassword(ctx context.Context, adminID, targetID uint64, ip, ua string) error {
	user, err := s.userRepo.FindByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("FindByID: %w", err)
	}
	if user == nil {
		return apperror.ErrNotFound
	}

	tempPassword, err := hash.GenerateTempPassword(8, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("GenerateTempPassword: %w", err)
	}

	hashedPass, err := hash.HashPassword(tempPassword)
	if err != nil {
		return fmt.Errorf("HashPassword: %w", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		uForUpdate, err := s.userRepo.FindByIDForUpdate(txCtx, targetID)
		if err != nil {
			return err
		}

		uForUpdate.PasswordHash = hashedPass
		uForUpdate.MustChangePassword = true

		if err := s.userRepo.UpdateUser(txCtx, uForUpdate); err != nil {
			return err
		}

		if err := s.sessionRepo.DeleteByUserID(txCtx, targetID); err != nil {
			s.logger.Warn("không thể xóa sessions khi reset password", zap.Error(err), zap.Uint64("userID", targetID))
		}
		return nil
	})

	if err != nil {
		s.logAudit(ctx, adminID, "ADMIN_RESET_PASSWORD", fmt.Sprint(targetID), ip, ua, "failure")
		return err
	}

	if err := s.sessionRepo.RevokeAllUserTokens(ctx, targetID, 7*24*time.Hour); err != nil {
		s.logger.Warn("không thể revoke token redis khi reset password", zap.Error(err), zap.Uint64("userID", targetID))
	}
	s.logAudit(ctx, adminID, "ADMIN_RESET_PASSWORD", fmt.Sprint(targetID), ip, ua, "success")

	go func() {
		if err := s.mailService.SendAdminResetPasswordEmail(user.Email, tempPassword); err != nil {
			s.logger.Error("Lỗi gửi mail mật khẩu tạm", zap.Error(err), zap.String("email", user.Email))
		}
	}()

	return nil
}

func (s *AdminUserService) NotifyUser(ctx context.Context, adminID, targetID uint64, req dto.AdminNotifyRequest, ip, ua string) error {
	user, err := s.userRepo.FindByID(ctx, targetID)

	if err != nil {
		return err
	}

	if user == nil {
		return apperror.ErrNotFound
	}

	if user.Status == constant.StatusInactive {
		return apperror.ErrBadRequest.WithMessage("chỉ có thể gửi thông báo cho tài khoản đang hoạt động ")
	}

	go func() {
		if err := s.mailService.SendAdminNotification(user.Email, req.Subject, req.Message); err != nil {
			s.logger.Error("lỗi gửi mail thông báo", zap.Error(err), zap.String("email", user.Email))
		}
	}()

	s.logAudit(ctx, adminID, "ADMIN_NOTIFY_USER", fmt.Sprint(targetID), ip, ua, "success")
	return nil

}
