package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/response"
	"github.com/quocdev03/user-access-management/pkg/validator"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func getUserID(c *gin.Context) (uint64, bool) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Error(c, apperror.ErrUnauthorized)
		return 0, false
	}
	userID, ok := userIDVal.(uint64)
	if !ok {
		response.Error(c, apperror.ErrUnauthorized)
		return 0, false
	}
	return userID, true
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	res, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Lấy thông tin hồ sơ thành công", res)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	req, ok := validator.BindAndValidate[dto.UpdateProfileRequest](c)
	if !ok {
		return
	}

	err := h.userService.UpdateProfile(c.Request.Context(), userID, *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Cập nhật hồ sơ thành công", nil)
}

func (h *UserHandler) RequestEmailChange(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	req, ok := validator.BindAndValidate[dto.RequestEmailChangeRequest](c)
	if !ok {
		return
	}

	err := h.userService.RequestEmailChange(c.Request.Context(), userID, *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Yêu cầu đổi email thành công. Mã xác thực OTP đã được gửi đến email cũ và email mới.", nil)
}

func (h *UserHandler) VerifyEmailChange(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	req, ok := validator.BindAndValidate[dto.VerifyEmailChangeRequest](c)
	if !ok {
		return
	}

	err := h.userService.VerifyEmailChange(c.Request.Context(), userID, *req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Thay đổi email thành công. Phiên hoạt động cũ của bạn đã được thu hồi.", nil)
}

func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	file, err := c.FormFile("avatar")
	if err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := h.userService.UploadAvatar(c.Request.Context(), userID, file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Tải ảnh đại diện lên thành công", res)
}

func (h *UserHandler) DeleteAvatar(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.userService.DeleteAvatar(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Xóa ảnh đại diện thành công", nil)
}

func (h *UserHandler) ResendChangeEmailOTP(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.userService.ResendChangeEmailOTP(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Đã gửi lại mã OTP. Vui lòng kiểm tra email.", nil)
}
