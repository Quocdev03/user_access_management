package model

import (
	"time"
)

type PasswordResetToken struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	IsUsed    bool      `json:"is_used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
