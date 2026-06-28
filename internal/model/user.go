package model

import "time"

// Trạng thái user
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusLocked   = "locked"
)

// User đại diện cho bảng users trong database
type User struct {
	ID                  uint64     `db:"id"`
	Username            string     `db:"username"`
	Email               string     `db:"email"`
	PasswordHash        string     `db:"password_hash"`
	FullName            string     `db:"full_name"`
	Phone               *string    `db:"phone"`         // Pointer vì có thể null
	AvatarURL           *string    `db:"avatar_url"`    // Pointer vì có thể null
	DateOfBirth         *string    `db:"date_of_birth"` // Pointer vì có thể null
	Status              string     `db:"status"`
	EmailVerified       bool       `db:"email_verified"`
	FailedLoginAttempts int        `db:"failed_login_attempts"`
	LockedUntil         *time.Time `db:"locked_until"`  // Pointer vì có thể null
	LastLoginAt         *time.Time `db:"last_login_at"` // Pointer vì có thể null
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}
