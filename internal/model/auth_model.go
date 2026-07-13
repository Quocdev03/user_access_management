package model

import "time"

type Session struct {
	ID               uint64    `db:"id" json:"id"`
	UserID           uint64    `db:"user_id" json:"user_id"`
	TokenHash        string    `db:"token_hash" json:"token_hash"`
	RefreshTokenHash string    `db:"refresh_token_hash" json:"refresh_token_hash"`
	JTI              *string   `db:"jti" json:"jti,omitempty"`
	IPAddress        *string   `db:"ip_address" json:"ip_address"`
	UserAgent        *string   `db:"user_agent" json:"user_agent"`
	DeviceID         *uint64   `db:"device_id" json:"device_id"`
	ExpiresAt        time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type Device struct {
	ID           uint64     `db:"id"`
	UserID       uint64     `db:"user_id"`
	DeviceName   *string    `db:"device_name"`
	DeviceType   *string    `db:"device_type"`
	OS           *string    `db:"os"`
	Browser      *string    `db:"browser"`
	IPAddress    *string    `db:"ip_address"`
	LastActiveAt *time.Time `db:"last_active_at"`
	CreatedAt    time.Time  `db:"created_at"`
}

type OTPCode struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	Code      string    `db:"code"`
	Type      string    `db:"type"`
	Attempts  int       `db:"attempts"`
	IsUsed    bool      `db:"is_used"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

type PasswordResetToken struct {
	ID        uint64    `db:"id" json:"id"`
	UserID    uint64    `db:"user_id" json:"user_id"`
	TokenHash string    `db:"token_hash" json:"token_hash"`
	IsUsed    bool      `db:"is_used" json:"is_used"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
