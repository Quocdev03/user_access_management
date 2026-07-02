package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
)

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsObj, exists := c.Get("tokenClaims")
		if !exists {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, ok := claimsObj.(*jwt.Claims)
		if !ok {
			response.Error(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range claims.Roles {
				if userRole == "admin" {
					hasRole = true
					break
				}
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			response.Error(c, apperror.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
