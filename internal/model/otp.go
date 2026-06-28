package model

import "time"

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
