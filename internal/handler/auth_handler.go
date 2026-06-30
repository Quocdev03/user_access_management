package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
)

// AuthHandler tiếp nhận các request HTTP liên quan đến phân hệ Authentication và gọi service tương ứng xử lý
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler khởi tạo một thể hiện mới của AuthHandler
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// bindAndValidate là hàm generic helper giúp giảm thiểu code lặp lại khi bind JSON.
func bindAndValidate[T any](c *gin.Context) (*T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return nil, false
	}
	return &req, true
}

// Register xử lý yêu cầu đăng ký tài khoản mới của người dùng
func (h *AuthHandler) Register(c *gin.Context) {
	req, ok := bindAndValidate[dto.RegisterRequest](c)
	if !ok {
		return
	}

	res, err := h.authService.Register(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Đăng ký thành công. Vui lòng kiểm tra email để nhận mã OTP.", res)
}

// VerifyEmail xác thực mã OTP được gửi về email của người dùng để kích hoạt tài khoản
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	req, ok := bindAndValidate[dto.VerifyEmailRequest](c)
	if !ok {
		return
	}

	err := h.authService.VerifyEmail(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Xác thực email thành công", nil)
}

// ResendVerificationEmail xử lý yêu cầu gửi lại mã OTP xác thực email
func (h *AuthHandler) ResendVerificationEmail(c *gin.Context) {
	req, ok := bindAndValidate[dto.ResendVerificationEmailRequest](c)
	if !ok {
		return
	}

	err := h.authService.ResendVerificationEmail(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đã gửi lại mã OTP. Vui lòng kiểm tra email của bạn.", nil)
}

// Login tiếp nhận thông tin đăng nhập bằng email, mật khẩu và trả về bộ đôi token (access/refresh)
func (h *AuthHandler) Login(c *gin.Context) {
	req, ok := bindAndValidate[dto.LoginRequest](c)
	if !ok {
		return
	}

	res, err := h.authService.Login(c.Request.Context(), *req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}

// RefreshToken xử lý việc gia hạn hoặc cấp lại access token bằng refresh token hợp lệ
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	req, ok := bindAndValidate[dto.RefreshTokenRequest](c)
	if !ok {
		return
	}

	res, err := h.authService.Refresh(c.Request.Context(), *req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Làm mới token thành công", res)
}

// Logout xử lý yêu cầu đăng xuất, thu hồi phiên đăng nhập hiện tại và đưa JTI của access token vào blacklist
func (h *AuthHandler) Logout(c *gin.Context) {
	rawToken := c.MustGet("token").(string)
	claims := c.MustGet("tokenClaims").(*jwt.Claims)

	err := h.authService.Logout(c.Request.Context(), rawToken, claims)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đăng xuất thành công", nil)
}

// ForgotPassword xử lý yêu cầu quên mật khẩu
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	req, ok := bindAndValidate[dto.ForgotPasswordRequest](c)
	if !ok {
		return
	}

	err := h.authService.ForgotPassword(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	// Luôn trả về thông báo chung để chống dò email
	response.Success(c, http.StatusOK, "Nếu email hợp lệ, hệ thống sẽ gửi một link khôi phục mật khẩu. Vui lòng kiểm tra hộp thư của bạn.", nil)
}

// ResetPassword xử lý đặt lại mật khẩu với token
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	req, ok := bindAndValidate[dto.ResetPasswordRequest](c)
	if !ok {
		return
	}

	err := h.authService.ResetPassword(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Khôi phục mật khẩu thành công. Vui lòng đăng nhập lại.", nil)
}

// ChangePassword xử lý thay đổi mật khẩu của user đang đăng nhập
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	req, ok := bindAndValidate[dto.ChangePasswordRequest](c)
	if !ok {
		return
	}

	claims := c.MustGet("tokenClaims").(*jwt.Claims)
	err := h.authService.ChangePassword(c.Request.Context(), claims.UserID, *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đổi mật khẩu thành công. Vui lòng đăng nhập lại trên các thiết bị.", nil)
}
