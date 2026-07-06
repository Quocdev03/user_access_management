package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/apperror"
)

type Response struct {
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Success bool        `json:"success"`
}

func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, statusCode int, message string, data interface{}, meta interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		errResp := gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
		}
		if appErr.Details != nil {
			errResp["details"] = appErr.Details
		}

		c.JSON(appErr.HTTPStatus, Response{
			Success: false,
			Error:   errResp,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Đã xảy ra lỗi hệ thống",
			"details": err.Error(), // Trong môi trường production, có thể ẩn đi
		},
	})
}

func ValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: gin.H{
			"code":    apperror.ErrValidationError.Code,
			"message": apperror.ErrValidationError.Message,
			"details": err.Error(),
		},
	})
}
