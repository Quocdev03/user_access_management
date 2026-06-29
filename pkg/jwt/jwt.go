// Thư viện xử lý JWT (JSON Web Token)
package jwt

import (
	"errors"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken báo lỗi khi token không hợp lệ hoặc chữ ký sai
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken báo lỗi khi token đã hết hạn sử dụng
	ErrExpiredToken = errors.New("token is expired")
)

// Claims định nghĩa các thông tin payload được lưu trữ trong JWT
type Claims struct {
	UserID uint64 `json:"sub"`  // ID của người dùng
	Type   string `json:"type"` // Loại token: "access" hoặc "refresh"
	golangjwt.RegisteredClaims   // Các claims chuẩn đăng ký sẵn theo RFC 7519
}

// GenerateToken sinh JWT token mới dựa trên ID người dùng, loại token, thời hạn và khóa bí mật
func GenerateToken(userID uint64, tokenType string, expiry time.Duration, secret string) (string, string, error) {
	jti := uuid.New().String() // Tạo mã ID duy nhất cho token (JTI)
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: golangjwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: golangjwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  golangjwt.NewNumericDate(now),
			NotBefore: golangjwt.NewNumericDate(now),
		},
	}

	token := golangjwt.NewWithClaims(golangjwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	return tokenString, jti, nil
}

// ParseToken thực hiện phân tích và xác thực tính hợp lệ của chuỗi JWT token
func ParseToken(tokenStr string, secret string) (*Claims, error) {
	token, err := golangjwt.ParseWithClaims(tokenStr, &Claims{}, func(token *golangjwt.Token) (interface{}, error) {
		// Kiểm tra thuật toán ký mã hóa token
		if _, ok := token.Method.(*golangjwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, golangjwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

