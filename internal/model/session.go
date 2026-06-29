package model

import "time"

// Session lưu trữ thông tin phiên đăng nhập của người dùng
type Session struct {
	ID               uint64    `db:"id" json:"id"`                               // ID duy nhất của phiên
	UserID           uint64    `db:"user_id" json:"user_id"`                     // ID của người dùng sở hữu phiên
	TokenHash        string    `db:"token_hash" json:"token_hash"`               // Hash SHA-256 của access token
	RefreshTokenHash string    `db:"refresh_token_hash" json:"refresh_token_hash"` // Hash SHA-256 của refresh token
	IPAddress        *string   `db:"ip_address" json:"ip_address"`               // Địa chỉ IP của thiết bị đăng nhập
	UserAgent        *string   `db:"user_agent" json:"user_agent"`               // Thông tin trình duyệt/thiết bị (User-Agent)
	DeviceID         *uint64   `db:"device_id" json:"device_id"`                 // Liên kết đến bảng thiết bị (nếu có)
	ExpiresAt        time.Time `db:"expires_at" json:"expires_at"`               // Thời điểm phiên hết hạn
	CreatedAt        time.Time `db:"created_at" json:"created_at"`               // Thời điểm tạo phiên
}
