package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/apperror"
)

// Response format chuẩn
type Response struct {
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Success bool        `json:"success"`
}

// Success returns a standardized success response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error handles application errors and formats them properly
func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, Response{
			Success: false,
			Error: gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		})
		return
	}

	// Fallback for unhandled internal errors
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: gin.H{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": "Đã xảy ra lỗi hệ thống",
			"details": err.Error(), // In production, consider hiding details based on env
		},
	})
}

// ValidationError handles request validation errors specifically
func ValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Dữ liệu đầu vào không hợp lệ",
			"details": err.Error(),
		},
	})
}
