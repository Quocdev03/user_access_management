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

// Register xử lý yêu cầu đăng ký tài khoản mới của người dùng
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Đăng ký thành công. Vui lòng kiểm tra email để nhận mã OTP.", res)
}

// VerifyEmail xác thực mã OTP được gửi về email của người dùng để kích hoạt tài khoản
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	err := h.authService.VerifyEmail(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Xác thực email thành công", nil)
}

// Login tiếp nhận thông tin đăng nhập bằng email, mật khẩu và trả về bộ đôi token (access/refresh)
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := h.authService.Login(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}

// RefreshToken xử lý việc gia hạn hoặc cấp lại access token bằng refresh token hợp lệ
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := h.authService.Refresh(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
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

