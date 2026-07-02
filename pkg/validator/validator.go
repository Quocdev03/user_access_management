package validator

import (
	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/response"
)

func BindAndValidate[T any](c *gin.Context) (*T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return nil, false
	}
	return &req, true
}
