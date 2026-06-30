package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
)


type AuthHandler struct {
	authService     *service.AuthService
	passwordService *service.PasswordService
}


func NewAuthHandler(authService *service.AuthService, passwordService *service.PasswordService) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		passwordService: passwordService,
	}
}


func bindAndValidate[T any](c *gin.Context) (*T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return nil, false
	}
	return &req, true
}


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


func (h *AuthHandler) LogoutAll(c *gin.Context) {
	claims := c.MustGet("tokenClaims").(*jwt.Claims)

	err := h.authService.LogoutAll(c.Request.Context(), claims.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đăng xuất khỏi tất cả thiết bị thành công", nil)
}


func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	req, ok := bindAndValidate[dto.ForgotPasswordRequest](c)
	if !ok {
		return
	}

	err := h.passwordService.ForgotPassword(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	// Luôn trả về thông báo chung để chống dò email
	response.Success(c, http.StatusOK, "Nếu email hợp lệ, hệ thống sẽ gửi một link khôi phục mật khẩu. Vui lòng kiểm tra hộp thư của bạn.", nil)
}


func (h *AuthHandler) ResetPassword(c *gin.Context) {
	req, ok := bindAndValidate[dto.ResetPasswordRequest](c)
	if !ok {
		return
	}

	err := h.passwordService.ResetPassword(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Khôi phục mật khẩu thành công. Vui lòng đăng nhập lại.", nil)
}


func (h *AuthHandler) ChangePassword(c *gin.Context) {
	req, ok := bindAndValidate[dto.ChangePasswordRequest](c)
	if !ok {
		return
	}

	claims := c.MustGet("tokenClaims").(*jwt.Claims)
	err := h.passwordService.ChangePassword(c.Request.Context(), claims.UserID, *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đổi mật khẩu thành công. Vui lòng đăng nhập lại trên các thiết bị.", nil)
}
