package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
)

// Middleware nÃ y pháº£i Ä‘Æ°á»£c Ä‘áº·t SAU AuthMiddleware vÃ¬ nÃ³ phá»¥ thuá»™c vÃ o JWT Claims.
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
				// Náº¿u user lÃ  admin thÃ¬ luÃ´n Ä‘Æ°á»£c phÃ©p bypass
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
