package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
	"github.com/quocdev03/user-access-management/pkg/validator"
)

type AdminUserHandler struct {
	adminUserService *service.AdminUserService
}

func NewAdminUserHandler(adminUserService *service.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{
		adminUserService: adminUserService,
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

func (h *AdminUserHandler) ListUsers(ctx *gin.Context) {
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

func (h *AdminUserHandler) GetUserDetail(c *gin.Context) {
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

func (h *AdminUserHandler) UpdateUser(ctx *gin.Context) {
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

func (h *AdminUserHandler) ChangeUserStatus(ctx *gin.Context) {
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

func (h *AdminUserHandler) ResetPassword(ctx *gin.Context) {
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

func (h *AdminUserHandler) NotifyUser(ctx *gin.Context) {
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
