package model

import (
	"time"

	"github.com/quocdev03/user-access-management/internal/constant"
)

type User struct {
	ID                  uint64              `db:"id"`
	Username            string              `db:"username"`
	Email               string              `db:"email"`
	PasswordHash        string              `db:"password_hash"`
	FullName            string              `db:"full_name"`
	Phone               string              `db:"phone"`
	AvatarURL           *string             `db:"avatar_url"`
	DateOfBirth         time.Time           `db:"date_of_birth"`
	Status              constant.UserStatus `db:"status"`
	MustChangePassword  bool                `db:"must_change_password" json:"must_change_password"`
	EmailVerified       bool                `db:"email_verified"`
	FailedLoginAttempts int                 `db:"failed_login_attempts"`
	LockedUntil         *time.Time          `db:"locked_until"`
	LastLoginAt         *time.Time          `db:"last_login_at"`
	CreatedAt           time.Time           `db:"created_at"`
	UpdatedAt           time.Time           `db:"updated_at"`
}
