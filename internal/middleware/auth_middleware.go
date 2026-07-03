package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/config"
	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
	"github.com/quocdev03/user-access-management/pkg/jwt"
	"github.com/quocdev03/user-access-management/pkg/response"
	"go.uber.org/zap"
)

func AuthMiddleware(cfg *config.Config, sessionRepo *repository.SessionRepository, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, apperror.ErrMissingAuthHeader)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Error(c, apperror.ErrInvalidAuthHeader)
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := jwt.ParseToken(tokenStr, cfg.JWT.Secret)
		if err != nil {
			errCode := "ERR_INVALID_TOKEN"
			errMsg := "Mã token xác thực không hợp lệ"
			if errors.Is(err, jwt.ErrExpiredToken) {
				errCode = "ERR_TOKEN_EXPIRED"
				errMsg = "Mã token xác thực đã hết hạn"
			}
			response.Error(c, apperror.NewAppError(errCode, errMsg, http.StatusUnauthorized))
			c.Abort()
			return
		}

		if claims.Type != "access" {
			response.Error(c, apperror.ErrNotAccessToken)
			c.Abort()
			return
		}

		blacklisted, err := sessionRepo.IsBlacklisted(c.Request.Context(), claims.ID)
		if err != nil {
			logger.Error("Lỗi kết nối Redis khi check blacklist, từ chối request", zap.Error(err), zap.String("jti", claims.ID))
			response.Error(c, apperror.NewAppError("ERR_SERVICE_UNAVAILABLE", "Hệ thống tạm thời không khả dụng", http.StatusServiceUnavailable))
			c.Abort()
			return
		}
		if blacklisted {
			response.Error(c, apperror.ErrTokenRevoked)
			c.Abort()
			return
		}

		revokedEpoch, err := sessionRepo.GetUserRevokedEpoch(c.Request.Context(), claims.UserID)
		if err != nil {
			logger.Error("Lỗi kết nối Redis khi check revoked epoch, từ chối request", zap.Error(err), zap.Uint64("user_id", claims.UserID))
			response.Error(c, apperror.NewAppError("ERR_SERVICE_UNAVAILABLE", "Hệ thống tạm thời không khả dụng", http.StatusServiceUnavailable))
			c.Abort()
			return
		}
		if revokedEpoch > 0 {
			if claims.IssuedAt.Unix() <= revokedEpoch {
				response.Error(c, apperror.ErrSessionRevokedGlobal)
				c.Abort()
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("token", tokenStr)
		c.Set("tokenClaims", claims)

		c.Next()
	}
}
