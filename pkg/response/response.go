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

// Success trả về dữ liệu response thành công theo định dạng chuẩn hóa
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error xử lý các lỗi ứng dụng và định dạng chúng phù hợp theo chuẩn hóa
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

	// Xử lý mặc định (Fallback) cho các lỗi nội bộ hệ thống chưa được phân loại
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: gin.H{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": "Đã xảy ra lỗi hệ thống",
			"details": err.Error(), // Trong môi trường production, có thể cân nhắc ẩn chi tiết này dựa trên cấu hình môi trường
		},
	})
}

// ValidationError xử lý riêng các lỗi kiểm tra tính hợp lệ dữ liệu đầu vào (Validation)
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
