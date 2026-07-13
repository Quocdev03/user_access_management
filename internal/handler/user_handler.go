package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/hash"
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

// @Summary Lấy thông tin cá nhân
// @Description Lấy thông tin hồ sơ của người dùng hiện tại
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.UserProfileResponse}
// @Failure 401 {object} response.Response
// @Router /users/me [get]
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

// @Summary Cập nhật hồ sơ
// @Description Cập nhật thông tin hồ sơ của người dùng hiện tại
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.UpdateProfileRequest true "Thông tin cập nhật"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me [patch]
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

// @Summary Yêu cầu đổi email
// @Description Yêu cầu đổi địa chỉ email, hệ thống sẽ gửi OTP
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.RequestEmailChangeRequest true "Email mới"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/email/change-request [post]
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

// @Summary Xác thực đổi email
// @Description Nhập mã OTP để hoàn tất quá trình đổi email
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.VerifyEmailChangeRequest true "OTP cũ và mới"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/email/verify [post]
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

// @Summary Tải lên ảnh đại diện
// @Description Upload ảnh đại diện mới
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "File ảnh"
// @Success 200 {object} response.Response{data=dto.UploadAvatarResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/avatar [post]
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		response.ValidationError(c, err)
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		response.Error(c, apperror.ErrBadRequest.WithMessage("Không thể mở tệp tải lên"))
		return
	}
	defer src.Close()

	res, err := h.userService.UploadAvatar(c.Request.Context(), userID, src, fileHeader.Filename, fileHeader.Size)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Tải ảnh đại diện lên thành công", res)
}

// @Summary Xóa ảnh đại diện
// @Description Xóa ảnh đại diện hiện tại
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/avatar [delete]
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

// @Summary Lấy danh sách phiên đăng nhập
// @Description Lấy danh sách các phiên đăng nhập đang hoạt động của người dùng
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]dto.SessionResponse}
// @Failure 401 {object} response.Response
// @Router /users/me/sessions [get]
func (h *UserHandler) GetSessions(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	tokenStr, exists := c.Get("token")
	var currentTokenHash string
	if exists {
		currentTokenHash = hash.SHA256(tokenStr.(string))
	}

	res, err := h.userService.GetSessions(c.Request.Context(), userID, currentTokenHash)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Thành công", res)
}

// @Summary Xóa phiên đăng nhập
// @Description Hủy một phiên đăng nhập cụ thể theo ID
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Session ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/sessions/{id} [delete]
func (h *UserHandler) RevokeSession(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	sessionID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	err := h.userService.RevokeSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Đã xóa phiên", nil)
}

// @Summary Lấy danh sách thiết bị
// @Description Lấy thông tin các thiết bị đã đăng nhập
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]dto.DeviceResponse}
// @Failure 401 {object} response.Response
// @Router /users/me/devices [get]
func (h *UserHandler) GetDevices(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	res, err := h.userService.GetDevices(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Thành công", res)
}

// @Summary Gửi lại OTP đổi email
// @Description Yêu cầu gửi lại OTP cho thao tác đổi email
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/email/resend [post]
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
