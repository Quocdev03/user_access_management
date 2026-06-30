package model

import "time"

type Session struct {
	ID               uint64    `db:"id" json:"id"`
	UserID           uint64    `db:"user_id" json:"user_id"`
	TokenHash        string    `db:"token_hash" json:"token_hash"`
	RefreshTokenHash string    `db:"refresh_token_hash" json:"refresh_token_hash"`
	IPAddress        *string   `db:"ip_address" json:"ip_address"`
	UserAgent        *string   `db:"user_agent" json:"user_agent"`
	DeviceID         *uint64   `db:"device_id" json:"device_id"`
	ExpiresAt        time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}
