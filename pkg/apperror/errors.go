package apperror

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	HTTPStatus int         `json:"-"`
	Details    interface{} `json:"details,omitempty"`
	RootErr    error       `json:"-"`
}

func (e *AppError) Error() string {
	if e.RootErr != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.RootErr)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.RootErr
}

func (e AppError) WithErr(err error) *AppError {
	e.RootErr = err
	return &e
}

func (e AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return &e
}

func (e AppError) WithMessage(msg string) *AppError {
	e.Message = msg
	return &e
}

func NewAppError(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

var (
	ErrBadRequest        = NewAppError("BAD_REQUEST", "Yêu cầu không hợp lệ", http.StatusBadRequest)
	ErrValidationError   = NewAppError("VALIDATION_ERROR", "Dữ liệu đầu vào không hợp lệ", http.StatusBadRequest)
	ErrInvalidParam      = NewAppError("INVALID_PARAM", "Tham số truyền vào không hợp lệ", http.StatusBadRequest)
	ErrInvalidDateFormat = NewAppError("INVALID_DATE_FORMAT", "Ngày sinh không đúng định dạng YYYY-MM-DD", http.StatusBadRequest)
	ErrResetTokenInvalid = NewAppError("INVALID_TOKEN", "Link khôi phục mật khẩu không hợp lệ hoặc đã hết hạn", http.StatusBadRequest)
	ErrResetTokenUsed    = NewAppError("INVALID_TOKEN", "Link khôi phục mật khẩu đã được sử dụng hoặc hết hạn", http.StatusBadRequest)
	ErrSamePassword      = NewAppError("SAME_PASSWORD", "Mật khẩu mới không được trùng với mật khẩu cũ", http.StatusBadRequest)

	ErrUnauthorized         = NewAppError("UNAUTHORIZED", "Vui lòng đăng nhập để tiếp tục", http.StatusUnauthorized)
	ErrInvalidCredentials   = NewAppError("INVALID_CREDENTIALS", "Email hoặc mật khẩu không chính xác", http.StatusUnauthorized)
	ErrTokenExpired         = NewAppError("TOKEN_EXPIRED", "Phiên đăng nhập đã hết hạn", http.StatusUnauthorized)
	ErrTokenInvalid         = NewAppError("TOKEN_INVALID", "Token không hợp lệ hoặc đã bị giả mạo", http.StatusUnauthorized)
	ErrSessionExpired       = NewAppError("SESSION_EXPIRED", "Phiên đăng nhập không tồn tại hoặc đã bị thu hồi", http.StatusUnauthorized)
	ErrMissingAuthHeader    = NewAppError("UNAUTHORIZED", "Thiếu header ủy quyền (Authorization header)", http.StatusUnauthorized)
	ErrInvalidAuthHeader    = NewAppError("INVALID_TOKEN", "Định dạng header ủy quyền không hợp lệ", http.StatusUnauthorized)
	ErrNotAccessToken       = NewAppError("INVALID_TOKEN", "Mã xác thực không phải là access token", http.StatusUnauthorized)
	ErrTokenRevoked         = NewAppError("TOKEN_REVOKED", "Mã token xác thực đã bị thu hồi", http.StatusUnauthorized)
	ErrSessionRevokedGlobal = NewAppError("TOKEN_REVOKED", "Phiên đăng nhập đã hết hạn do tài khoản bị đăng xuất khỏi tất cả thiết bị", http.StatusUnauthorized)
	ErrRefreshTokenInvalid  = NewAppError("REFRESH_INVALID", "Mã refresh token không hợp lệ hoặc đã hết hạn", http.StatusUnauthorized)
	ErrNotRefreshToken      = NewAppError("REFRESH_INVALID", "Mã token không phải refresh token", http.StatusUnauthorized)
	ErrTokenReuse           = NewAppError("TOKEN_REUSE", "Phát hiện truy cập bất thường, vui lòng đăng nhập lại", http.StatusUnauthorized)

	ErrForbidden              = NewAppError("FORBIDDEN", "Bạn không có quyền thực hiện thao tác này", http.StatusForbidden)
	ErrAccountInactive        = NewAppError("INACTIVE_ACCOUNT", "Vui lòng xác thực email trước khi đăng nhập", http.StatusForbidden)
	ErrAccountAlreadyVerified = NewAppError("ALREADY_VERIFIED", "Tài khoản đã được xác thực, vui lòng đăng nhập", http.StatusForbidden)
	ErrAccountDisabled        = NewAppError("ACCOUNT_DISABLED", "Tài khoản của bạn đã bị khóa hoặc vô hiệu hóa", http.StatusForbidden)

	ErrNotFound      = NewAppError("NOT_FOUND", "Không tìm thấy dữ liệu yêu cầu", http.StatusNotFound)
	ErrEmailNotFound = NewAppError("EMAIL_NOT_FOUND", "Email chưa được đăng ký", http.StatusNotFound)
	ErrConflict      = NewAppError("CONFLICT", "Dữ liệu đã tồn tại hoặc bị xung đột", http.StatusConflict)
	ErrAccountLocked = NewAppError("ACCOUNT_LOCKED", "Tài khoản đang bị khóa, không thể thực hiện thao tác này", http.StatusLocked)

	ErrOTPExpired     = NewAppError("OTP_EXPIRED", "Không tìm thấy mã OTP hợp lệ hoặc đã hết hạn", http.StatusBadRequest)
	ErrOTPInvalid     = NewAppError("OTP_INVALID", "Mã OTP không đúng", http.StatusBadRequest)
	ErrOTPMaxAttempts = NewAppError("OTP_MAX_ATTEMPTS", "Bạn đã nhập sai OTP quá 5 lần, vui lòng yêu cầu mã mới", http.StatusBadRequest)

	ErrFileTooLarge    = NewAppError("FILE_TOO_LARGE", "Dung lượng file vượt quá giới hạn cho phép", http.StatusRequestEntityTooLarge)
	ErrInvalidFileType = NewAppError("INVALID_FILE_TYPE", "Định dạng file không được hỗ trợ", http.StatusUnsupportedMediaType)

	ErrRateLimited       = NewAppError("RATE_LIMITED", "Bạn thao tác quá nhanh, vui lòng thử lại sau", http.StatusTooManyRequests)
	ErrRateLimitedMinute = NewAppError("RATE_LIMITED", "Bạn thao tác quá nhanh, vui lòng chờ 1 phút trước khi yêu cầu lại", http.StatusTooManyRequests)
	ErrIPBanned          = NewAppError("IP_BANNED", "Địa chỉ IP của bạn đã bị khóa tạm thời do phát hiện hành vi spam", http.StatusForbidden)

	ErrInternalServer = NewAppError("INTERNAL_ERROR", "Đã xảy ra lỗi hệ thống, vui lòng thử lại sau", http.StatusInternalServerError)
	ErrDatabase       = NewAppError("DATABASE_ERROR", "Lỗi thao tác cơ sở dữ liệu", http.StatusInternalServerError)
)
