package middleware

import (
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

// AuthMiddleware kiểm tra và xác thực access token của người dùng trên các route yêu cầu bảo mật
func AuthMiddleware(cfg *config.Config, sessionRepo repository.SessionRepository, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy Authorization header từ request
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, apperror.NewAppError("ERR_UNAUTHORIZED", "Thiếu header ủy quyền (Authorization header)", http.StatusUnauthorized))
			c.Abort()
			return
		}

		// Định dạng token phải là: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Error(c, apperror.NewAppError("ERR_INVALID_TOKEN", "Định dạng header ủy quyền không hợp lệ", http.StatusUnauthorized))
			c.Abort()
			return
		}

		tokenStr := parts[1]
		// Phân tích cú pháp và xác thực chữ ký token
		claims, err := jwt.ParseToken(tokenStr, cfg.JWT.Secret)
		if err != nil {
			var errMsg string
			var errCode string
			if err == jwt.ErrExpiredToken {
				errCode = "ERR_TOKEN_EXPIRED"
				errMsg = "Mã token xác thực đã hết hạn"
			} else {
				errCode = "ERR_INVALID_TOKEN"
				errMsg = "Mã token xác thực không hợp lệ"
			}
			response.Error(c, apperror.NewAppError(errCode, errMsg, http.StatusUnauthorized))
			c.Abort()
			return
		}

		// Kiểm tra loại token (phải là access token)
		if claims.Type != "access" {
			response.Error(c, apperror.NewAppError("ERR_INVALID_TOKEN", "Mã xác thực không phải là access token", http.StatusUnauthorized))
			c.Abort()
			return
		}

		// Kiểm tra danh sách thu hồi token (blacklist) trên Redis. Áp dụng cơ chế Fail-Open
		blacklisted, err := sessionRepo.IsBlacklisted(c.Request.Context(), claims.ID)
		if err != nil {
			// Fail-Open: Ghi nhận cảnh báo khi Redis lỗi nhưng không chặn người dùng hợp lệ
			logger.Error("Lỗi kết nối Redis khi check blacklist, bỏ qua kiểm tra", zap.Error(err), zap.String("jti", claims.ID))
		} else if blacklisted {
			response.Error(c, apperror.NewAppError("ERR_TOKEN_REVOKED", "Mã token xác thực đã bị thu hồi", http.StatusUnauthorized))
			c.Abort()
			return
		}

		// Lưu trữ các thông tin hữu ích vào context của request cho các handler phía sau sử dụng
		c.Set("userID", claims.UserID)
		c.Set("jti", claims.ID)
		c.Set("token", tokenStr)
		c.Set("tokenClaims", claims)

		c.Next()
	}
}
