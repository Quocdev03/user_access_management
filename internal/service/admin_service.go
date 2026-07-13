package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/quocdev03/user-access-management/internal/config"
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
	cfg          *config.Config
	logger       *zap.Logger
}

func NewAdminUserService(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	auditLogRepo *repository.AuditLogRepository,
	mailService *MailService,
	txManager *database.TxManager,
	cfg *config.Config,
	logger *zap.Logger,
) *AdminUserService {
	return &AdminUserService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		sessionRepo:  sessionRepo,
		auditLogRepo: auditLogRepo,
		mailService:  mailService,
		txManager:    txManager,
		cfg:          cfg,
		logger:       logger,
	}
}

func (s *AdminUserService) logAudit(ctx context.Context, adminID uint64, action, resourceID, ip, ua, status string) {
	resource := "users"
	if err := s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:     &adminID,
		Action:     action,
		Resource:   &resource,
		ResourceID: &resourceID,
		IPAddress:  &ip,
		UserAgent:  &ua,
		Status:     status,
	}); err != nil {
		s.logger.Warn("Không thể ghi audit log", zap.String("action", action), zap.Error(err))
	}
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
		req.PerPage,
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

func (s *AdminUserService) UpdateUser(
	ctx context.Context,
	adminID, targetID uint64,
	req dto.AdminUpdateUserRequest,
	ip, ua string,
) error {
	var needsRevokeSession bool

	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		user, err := s.userRepo.FindByIDForUpdate(txCtx, targetID)
		if err != nil {
			return err
		}
		if user == nil {
			return apperror.ErrNotFound
		}

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

		return nil
	})

	if err != nil {
		s.logAudit(ctx, adminID, "ADMIN_UPDATE_USER", fmt.Sprint(targetID), ip, ua, "failure")
		return err
	}

	if needsRevokeSession {
		if err := s.sessionRepo.InvalidateUserSessions(ctx, targetID, s.cfg.JWT.RefreshExpiry); err != nil {
			s.logger.Warn("không thể invalidate sessions sau update user", zap.Error(err), zap.Uint64("userID", targetID))
		}
	}

	s.logAudit(ctx, adminID, "ADMIN_UPDATE_USER", fmt.Sprint(targetID), ip, ua, "success")
	return nil
}

func (s *AdminUserService) ChangeUserStatus(
	ctx context.Context,
	adminID, targetID uint64,
	req dto.AdminChangeStatusRequest,
	ip, ua string,
) error {
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
		return s.userRepo.UpdateUser(txCtx, user)
	})

	if err != nil {
		s.logAudit(ctx, adminID, "ADMIN_CHANGE_STATUS", fmt.Sprint(targetID), ip, ua, "failure")
		return err
	}

	if req.Status == string(constant.StatusLocked) || req.Status == string(constant.StatusInactive) {
		if err := s.sessionRepo.InvalidateUserSessions(ctx, targetID, s.cfg.JWT.RefreshExpiry); err != nil {
			s.logger.Warn("không thể invalidate sessions khi đổi trạng thái user", zap.Error(err), zap.Uint64("userID", targetID))
		}
	}

	s.logAudit(ctx, adminID, "ADMIN_CHANGE_STATUS", fmt.Sprint(targetID), ip, ua, "success")
	return nil
}

func (s *AdminUserService) ResetUserPassword(
	ctx context.Context,
	adminID, targetID uint64,
	ip, ua string,
) error {
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

		return s.userRepo.UpdateUser(txCtx, uForUpdate)
	})

	if err != nil {
		s.logAudit(ctx, adminID, "ADMIN_RESET_PASSWORD", fmt.Sprint(targetID), ip, ua, "failure")
		return err
	}

	if err := s.sessionRepo.InvalidateUserSessions(ctx, targetID, s.cfg.JWT.RefreshExpiry); err != nil {
		s.logger.Warn("không thể invalidate sessions khi reset password", zap.Error(err), zap.Uint64("userID", targetID))
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

type AdminRoleService struct {
	roleRepo       *repository.RoleRepository
	permissionRepo *repository.PermissionRepository
	sessionRepo    *repository.SessionRepository
	txManager      *database.TxManager
	cfg            *config.Config
	logger         *zap.Logger
}

func NewAdminRoleService(
	roleRepo *repository.RoleRepository,
	permissionRepo *repository.PermissionRepository,
	sessionRepo *repository.SessionRepository,
	txManager *database.TxManager,
	cfg *config.Config,
	logger *zap.Logger,
) *AdminRoleService {
	return &AdminRoleService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		sessionRepo:    sessionRepo,
		txManager:      txManager,
		cfg:            cfg,
		logger:         logger,
	}
}

func (s *AdminRoleService) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	roles, err := s.roleRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	res := []dto.RoleResponse{}
	for _, r := range roles {
		perms, _ := s.roleRepo.GetPermissionsByRoleID(ctx, r.ID)
		pRes := []dto.PermissionResponse{}
		for _, p := range perms {
			pRes = append(pRes, dto.PermissionResponse{
				ID: p.ID, Name: p.Name, Description: p.Description, Resource: p.Resource, Action: p.Action,
			})
		}
		res = append(res, dto.RoleResponse{
			ID: r.ID, Name: r.Name, Description: r.Description, CreatedAt: r.CreatedAt, Permissions: pRes,
		})
	}
	return res, nil
}

func (s *AdminRoleService) CreateRole(ctx context.Context, req dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	role := &model.Role{Name: req.Name, Description: req.Description}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return &dto.RoleResponse{ID: role.ID, Name: role.Name, Description: role.Description, CreatedAt: role.CreatedAt}, nil
}

func (s *AdminRoleService) UpdateRole(ctx context.Context, id uint64, req dto.UpdateRoleRequest) error {
	return s.roleRepo.Update(ctx, &model.Role{ID: id, Name: req.Name, Description: req.Description})
}

func (s *AdminRoleService) DeleteRole(ctx context.Context, id uint64) error {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("roleRepo.FindByID: %w", err)
	}
	if role == nil {
		return apperror.ErrNotFound
	}

	switch role.Name {
	case constant.RoleAdmin, constant.RoleModerator, constant.RoleUser:
		return apperror.ErrBadRequest.WithMessage("Không thể xóa role hệ thống")
	}

	count, err := s.roleRepo.CountUsersByRoleID(ctx, id)
	if err != nil {
		return fmt.Errorf("roleRepo.CountUsersByRoleID: %w", err)
	}
	if count > 0 {
		return apperror.ErrBadRequest.WithMessage("Không thể xóa role đang được gán cho người dùng")
	}

	return s.roleRepo.Delete(ctx, id)
}

func (s *AdminRoleService) AssignPermissions(ctx context.Context, roleID uint64, req dto.AssignPermissionsRequest) error {
	if len(req.PermissionIDs) > 0 {
		count, err := s.permissionRepo.CountByIDs(ctx, req.PermissionIDs)
		if err != nil {
			return fmt.Errorf("permissionRepo.CountByIDs: %w", err)
		}
		if count != len(req.PermissionIDs) {
			return apperror.ErrBadRequest.WithMessage("Một hoặc nhiều permission_id không tồn tại")
		}
	}
	return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		return s.roleRepo.AssignPermissions(txCtx, roleID, req.PermissionIDs)
	})
}

func (s *AdminRoleService) AssignUserRole(ctx context.Context, userID uint64, req dto.AssignRoleRequest) error {
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		return s.roleRepo.AssignRoleToUser(txCtx, userID, req.RoleID)
	})
	if err != nil {
		return err
	}
	if err := s.sessionRepo.InvalidateUserSessions(ctx, userID, s.cfg.JWT.RefreshExpiry); err != nil {
		return fmt.Errorf("sessionRepo.InvalidateUserSessions: %w", err)
	}
	return nil
}

func (s *AdminRoleService) RemoveUserRole(ctx context.Context, userID, roleID uint64) error {
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		return s.roleRepo.RemoveRoleFromUser(txCtx, userID, roleID)
	})
	if err != nil {
		return err
	}
	if err := s.sessionRepo.InvalidateUserSessions(ctx, userID, s.cfg.JWT.RefreshExpiry); err != nil {
		return fmt.Errorf("sessionRepo.InvalidateUserSessions: %w", err)
	}
	return nil
}

type AdminAuditLogService struct {
	auditLogRepo *repository.AuditLogRepository
	logger       *zap.Logger
}

func NewAdminAuditLogService(auditLogRepo *repository.AuditLogRepository, logger *zap.Logger) *AdminAuditLogService {
	return &AdminAuditLogService{auditLogRepo: auditLogRepo, logger: logger}
}

func (s *AdminAuditLogService) ExportAuditLogs(ctx context.Context, req dto.ExportAuditLogsRequest) ([]byte, error) {
	logs, err := s.auditLogRepo.FindAllWithFilters(
		ctx,
		req.UserID,
		req.Action,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("auditLogRepo.FindAllWithFilters: %w", err)
	}

	var buf bytes.Buffer
	// Đính kèm BOM để Excel hiển thị tốt Tiếng Việt
	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(&buf)
	header := []string{"ID", "UserID", "Action", "Resource", "ResourceID", "IPAddress", "Status", "CreatedAt"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, log := range logs {
		userID := ""
		if log.UserID != nil {
			userID = fmt.Sprint(*log.UserID)
		}
		resource := ""
		if log.Resource != nil {
			resource = *log.Resource
		}
		resourceID := ""
		if log.ResourceID != nil {
			resourceID = *log.ResourceID
		}
		ip := ""
		if log.IPAddress != nil {
			ip = *log.IPAddress
		}

		row := []string{
			fmt.Sprint(log.ID), userID, log.Action, resource, resourceID, ip, log.Status,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		_ = writer.Write(row)
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}
