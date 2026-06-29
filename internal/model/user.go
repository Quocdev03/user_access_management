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
	Phone               string     `db:"phone"`
	AvatarURL           *string    `db:"avatar_url"`
	DateOfBirth         time.Time  `db:"date_of_birth"`
	Status              string     `db:"status"`
	EmailVerified       bool       `db:"email_verified"`
	FailedLoginAttempts int        `db:"failed_login_attempts"`
	LockedUntil         *time.Time `db:"locked_until"` // Sử dụng con trỏ vì trường này có thể mang giá trị NULL trong cơ sở dữ liệu
	LastLoginAt         *time.Time `db:"last_login_at"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}
