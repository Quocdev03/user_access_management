package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
	"github.com/quocdev03/user-access-management/pkg/validator"
)

type AdminHandler struct {
	adminUserService *service.AdminUserService
	roleService      *service.AdminRoleService
	auditLogService  *service.AdminAuditLogService
}

func NewAdminHandler(u *service.AdminUserService, r *service.AdminRoleService, a *service.AdminAuditLogService) *AdminHandler {
	return &AdminHandler{
		adminUserService: u,
		roleService:      r,
		auditLogService:  a,
	}
}

func parseTargetID(ctx *gin.Context) (uint64, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, apperror.ErrBadRequest.WithMessage("ID người dùng không hợp lệ")
	}
	return id, nil
}

// @Summary Lấy danh sách người dùng
// @Description Dành cho Admin lấy danh sách users (hỗ trợ phân trang, lọc, sắp xếp)
// @Tags Admin - Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Số trang"
// @Param page_size query int false "Kích thước trang"
// @Param sort_by query string false "Sắp xếp theo trường nào"
// @Param sort_order query string false "Chiều sắp xếp (asc/desc)"
// @Param status query string false "Lọc theo trạng thái"
// @Param search query string false "Tìm kiếm theo username/email"
// @Success 200 {object} response.Response{data=[]dto.UserProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users [get]
func (h *AdminHandler) ListUsers(ctx *gin.Context) {
	var req dto.AdminListUsersRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		response.Error(ctx, apperror.ErrBadRequest.WithMessage("query parameters không hợp lệ"))
		return
	}

	res, err := h.adminUserService.ListUsers(ctx.Request.Context(), req)
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.SuccessWithMeta(ctx, http.StatusOK, "lấy danh sách thành công", res.Users, res.Meta)
}

// @Summary Xem chi tiết người dùng
// @Description Dành cho Admin xem chi tiết thông tin của một user cụ thể
// @Tags Admin - Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Response{data=dto.UserProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id} [get]
func (h *AdminHandler) GetUserDetail(c *gin.Context) {
	targetID, err := parseTargetID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	res, err := h.adminUserService.GetUserDetail(c.Request.Context(), targetID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Lấy thông tin thành công", res)
}

// @Summary Cập nhật thông tin người dùng
// @Description Dành cho Admin cập nhật thông tin cá nhân của user
// @Tags Admin - Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body dto.AdminUpdateUserRequest true "Dữ liệu cập nhật"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(ctx *gin.Context) {
	targetID, err := parseTargetID(ctx)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	claims := ctx.MustGet("tokenClaims").(*jwt.Claims)
	req, ok := validator.BindAndValidate[dto.AdminUpdateUserRequest](ctx)
	if !ok {
		return
	}

	err = h.adminUserService.UpdateUser(
		ctx.Request.Context(),
		claims.UserID,
		targetID,
		*req,
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.Success(ctx, http.StatusOK, "Cập nhật người dùng thành công", nil)
}

// @Summary Thay đổi trạng thái người dùng
// @Description Dành cho Admin khóa hoặc mở khóa tài khoản user
// @Tags Admin - Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body dto.AdminChangeStatusRequest true "Trạng thái và lý do"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/status [patch]
func (h *AdminHandler) ChangeUserStatus(ctx *gin.Context) {
	targetID, err := parseTargetID(ctx)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	claims := ctx.MustGet("tokenClaims").(*jwt.Claims)
	req, ok := validator.BindAndValidate[dto.AdminChangeStatusRequest](ctx)
	if !ok {
		return
	}

	err = h.adminUserService.ChangeUserStatus(
		ctx.Request.Context(),
		claims.UserID,
		targetID,
		*req,
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	)

	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.Success(ctx, http.StatusOK, "cập nhật trạng thái thành công", nil)
}

// @Summary Đặt lại mật khẩu người dùng
// @Description Dành cho Admin reset mật khẩu của user về mật khẩu tạm thời
// @Tags Admin - Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/reset-password [post]
func (h *AdminHandler) ResetPassword(ctx *gin.Context) {
	targetID, err := parseTargetID(ctx)
	if err != nil {
		response.Error(ctx, err)
	}

	claím := ctx.MustGet("tokenClaims").(*jwt.Claims)

	err = h.adminUserService.ResetUserPassword(
		ctx.Request.Context(),
		claím.UserID,
		targetID,
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	)

	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.Success(ctx, http.StatusOK, "đặt lại mật khẩu thành công. Mật khẩu tạm được gửi qua email", nil)
}

// @Summary Gửi thông báo cho người dùng
// @Description Dành cho Admin gửi email thông báo trực tiếp tới user
// @Tags Admin - Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body dto.AdminNotifyRequest true "Nội dung thông báo"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/notify [post]
func (h *AdminHandler) NotifyUser(ctx *gin.Context) {
	targetID, err := parseTargetID(ctx)
	if err != nil {
		response.Error(ctx, err)
	}
	claims := ctx.MustGet("tokenClaims").(*jwt.Claims)
	req, ok := validator.BindAndValidate[dto.AdminNotifyRequest](ctx)
	if !ok {
		return
	}

	err = h.adminUserService.NotifyUser(
		ctx.Request.Context(),
		claims.UserID,
		targetID,
		*req,
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	)
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.Success(ctx, http.StatusOK, "Đã gửi thông báo thành công", nil)
}

// @Summary Lấy danh sách Roles
// @Description Lấy danh sách tất cả các role trong hệ thống
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]dto.RoleResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/roles [get]
func (h *AdminHandler) ListRoles(c *gin.Context) {
	res, err := h.roleService.ListRoles(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Thành công", res)
}

// @Summary Tạo Role mới
// @Description Tạo một role mới trong hệ thống
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateRoleRequest true "Thông tin Role"
// @Success 200 {object} response.Response{data=dto.RoleResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/roles [post]
func (h *AdminHandler) CreateRole(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.CreateRoleRequest](c)
	if !ok {
		return
	}
	res, err := h.roleService.CreateRole(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Tạo role thành công", res)
}

// @Summary Cập nhật Role
// @Description Cập nhật thông tin của một role
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param request body dto.UpdateRoleRequest true "Thông tin cập nhật"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/roles/{id} [put]
func (h *AdminHandler) UpdateRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	req, ok := validator.BindAndValidate[dto.UpdateRoleRequest](c)
	if !ok {
		return
	}
	err := h.roleService.UpdateRole(c.Request.Context(), id, *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Cập nhật role thành công", nil)
}

// @Summary Xóa Role
// @Description Xóa một role khỏi hệ thống (nếu không có user đang sử dụng)
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/roles/{id} [delete]
func (h *AdminHandler) DeleteRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	err := h.roleService.DeleteRole(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Xóa role thành công", nil)
}

// @Summary Gán Permissions cho Role
// @Description Phân quyền (permissions) cho một role
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param request body dto.AssignPermissionsRequest true "Danh sách permission IDs"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/roles/{id}/permissions [post]
func (h *AdminHandler) AssignPermissions(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	req, ok := validator.BindAndValidate[dto.AssignPermissionsRequest](c)
	if !ok {
		return
	}
	err := h.roleService.AssignPermissions(c.Request.Context(), id, *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Gán quyền thành công", nil)
}

// @Summary Gán Role cho User
// @Description Gán một role cho một user
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body dto.AssignRoleRequest true "Role ID cần gán"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/roles [post]
func (h *AdminHandler) AssignUserRole(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	req, ok := validator.BindAndValidate[dto.AssignRoleRequest](c)
	if !ok {
		return
	}
	err := h.roleService.AssignUserRole(c.Request.Context(), userID, *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Gán role cho user thành công", nil)
}

// @Summary Gỡ Role khỏi User
// @Description Gỡ một role đã được gán cho một user
// @Tags Admin - Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param roleId path int true "Role ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/roles/{roleId} [delete]
func (h *AdminHandler) RemoveUserRole(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	roleID, _ := strconv.ParseUint(c.Param("roleId"), 10, 64)
	err := h.roleService.RemoveUserRole(c.Request.Context(), userID, roleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Gỡ role khỏi user thành công", nil)
}

// @Summary Xuất Audit Logs
// @Description Xuất nhật ký hệ thống (audit logs) ra file CSV (tối đa 10000 dòng)
// @Tags Admin - Audit Logs
// @Accept json
// @Produce text/csv
// @Security BearerAuth
// @Param user_id query int false "Lọc theo User ID"
// @Param action query string false "Lọc theo hành động (VD: login, create_user)"
// @Param start_date query string false "Từ ngày (YYYY-MM-DD)"
// @Param end_date query string false "Đến ngày (YYYY-MM-DD)"
// @Success 200 {file} file "File CSV"
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/audit-logs/export [get]
func (h *AdminHandler) Export(c *gin.Context) {
	var req dto.ExportAuditLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	csvData, err := h.auditLogService.ExportAuditLogs(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csvData)
}
