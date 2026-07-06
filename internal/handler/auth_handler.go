package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/dto"
	"github.com/quocdev03/user-access-management/internal/service"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
	"github.com/quocdev03/user-access-management/pkg/validator"
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


// @Summary Đăng ký tài khoản mới
// @Description Đăng ký người dùng mới vào hệ thống
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Thông tin đăng ký"
// @Success 201 {object} response.Response{data=dto.RegisterResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.RegisterRequest](c)
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


// @Summary Xác thực email
// @Description Xác thực tài khoản qua email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.VerifyEmailRequest true "OTP xác thực"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.VerifyEmailRequest](c)
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


// @Summary Gửi lại OTP xác thực
// @Description Yêu cầu gửi lại email chứa OTP xác thực tài khoản
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationEmailRequest true "Thông tin yêu cầu"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/resend-verification-email [post]
func (h *AuthHandler) ResendVerificationEmail(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.ResendVerificationEmailRequest](c)
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


// @Summary Đăng nhập
// @Description Đăng nhập vào hệ thống
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Thông tin đăng nhập"
// @Success 200 {object} response.Response{data=dto.LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.LoginRequest](c)
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


// @Summary Làm mới Access Token
// @Description Cấp lại Access Token mới dựa trên Refresh Token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} response.Response{data=dto.LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.RefreshTokenRequest](c)
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


// @Summary Đăng xuất
// @Description Đăng xuất khỏi hệ thống
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/logout [post]
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


// @Summary Đăng xuất khỏi tất cả thiết bị
// @Description Hủy tất cả các phiên đăng nhập của người dùng
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	claims := c.MustGet("tokenClaims").(*jwt.Claims)

	err := h.authService.LogoutAll(c.Request.Context(), claims.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đăng xuất khỏi tất cả thiết bị thành công", nil)
}


// @Summary Quên mật khẩu
// @Description Yêu cầu đặt lại mật khẩu bằng cách gửi email có chứa OTP
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email đã đăng ký"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.ForgotPasswordRequest](c)
	if !ok {
		return
	}

	err := h.passwordService.ForgotPassword(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Nếu email hợp lệ, hệ thống sẽ gửi một link khôi phục mật khẩu. Vui lòng kiểm tra hộp thư của bạn.", nil)
}


// @Summary Đặt lại mật khẩu
// @Description Đặt lại mật khẩu bằng OTP
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Thông tin đặt lại mật khẩu"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.ResetPasswordRequest](c)
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


// @Summary Đổi mật khẩu
// @Description Đổi mật khẩu tài khoản
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ChangePasswordRequest true "Thông tin đổi mật khẩu"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.ChangePasswordRequest](c)
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

// @Summary Đổi mật khẩu bắt buộc
// @Description Đổi mật khẩu (dành cho người dùng bị reset pass từ admin)
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ForceChangePasswordRequest true "Thông tin mật khẩu mới"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /auth/force-change-password [post]
func (h *AuthHandler) ForceChangePassword(c *gin.Context) {
	req, ok := validator.BindAndValidate[dto.ForceChangePasswordRequest](c)
	if !ok {
		return
	}

	err := h.passwordService.ForceChangePassword(c.Request.Context(), *req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Đổi mật khẩu bắt buộc thành công. Vui lòng đăng nhập lại với mật khẩu mới.", nil)
}
