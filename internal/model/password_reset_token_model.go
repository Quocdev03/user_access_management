package model

import (
	"time"
)

type PasswordResetToken struct {
	ID        uint64    `db:"id" json:"id"`
	UserID    uint64    `db:"user_id" json:"user_id"`
	TokenHash string    `db:"token_hash" json:"token_hash"`
	IsUsed    bool      `db:"is_used" json:"is_used"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
