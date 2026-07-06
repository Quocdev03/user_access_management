package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/constant"
	"github.com/quocdev03/user-access-management/internal/repository"
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

		reqMap := make(map[string]struct{}, len(roles))
		for _, r := range roles {
			reqMap[r] = struct{}{}
		}

		hasRole := false
		for _, userRole := range claims.Roles {
			if userRole == constant.RoleAdmin {
				hasRole = true
				break
			}
			if _, exists := reqMap[userRole]; exists {
				hasRole = true
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

func PermissionMiddleware(roleRepo *repository.RoleRepository, requiredPermission string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		val, exists := ctx.Get("tokenClaims")
		if !exists {
			response.Error(ctx, apperror.ErrUnauthorized)
			ctx.Abort()
			return
		}

		claims, ok := val.(*jwt.Claims)
		if !ok {
			response.Error(ctx, apperror.ErrUnauthorized)
			ctx.Abort()
			return
		}

		for _, role := range claims.Roles {
			if role == constant.RoleAdmin {
				ctx.Next()
				return
			}
		}

		permissions, err := roleRepo.GetPermissionsByUserId(ctx.Request.Context(), claims.UserID)
		if err != nil {
			response.Error(ctx, apperror.ErrInternalServer.WithMessage("không thể lấy dữ liệu phân quyền"))
			ctx.Abort()
			return
		}

		hasPerm := false
		for _, perm := range permissions {
			if perm == requiredPermission {
				hasPerm = true
				break
			}
		}

		if !hasPerm {
			response.Error(ctx, apperror.ErrForbidden)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
