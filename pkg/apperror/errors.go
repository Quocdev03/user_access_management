package apperror

import (
	"fmt"
	"net/http"
)

// AppError represents a standardized application error
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError creates a new AppError
func NewAppError(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Common Sentinel Errors
var (
	ErrInvalidCredentials = NewAppError("INVALID_CREDENTIALS", "Sai email hoặc mật khẩu", http.StatusUnauthorized)
	ErrAccountLocked      = NewAppError("ACCOUNT_LOCKED", "Tài khoản đã bị khóa", http.StatusLocked)
	ErrNotFound           = NewAppError("NOT_FOUND", "Không tìm thấy tài nguyên", http.StatusNotFound)
	ErrConflict           = NewAppError("CONFLICT", "Dữ liệu đã tồn tại", http.StatusConflict)
	ErrForbidden          = NewAppError("FORBIDDEN", "Không có quyền truy cập", http.StatusForbidden)
	ErrInternalServer     = NewAppError("INTERNAL_SERVER_ERROR", "Đã xảy ra lỗi hệ thống", http.StatusInternalServerError)
	ErrBadRequest         = NewAppError("BAD_REQUEST", "Dữ liệu yêu cầu không hợp lệ", http.StatusBadRequest)
)
