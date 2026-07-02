// Thư viện xử lý JWT (JSON Web Token)
package jwt

import (
	"errors"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token is expired")
)

type Claims struct {
	UserID uint64   `json:"sub"`
	Type   string   `json:"type"`
	Roles  []string `json:"roles"`
	golangjwt.RegisteredClaims
}

func GenerateToken(userID uint64, roles []string, tokenType string, expiry time.Duration, secret string) (string, string, error) {
	jti := uuid.New().String()
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   tokenType,
		Roles:  roles,
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

func ParseToken(tokenStr string, secret string) (*Claims, error) {
	token, err := golangjwt.ParseWithClaims(tokenStr, &Claims{}, func(token *golangjwt.Token) (interface{}, error) {
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
