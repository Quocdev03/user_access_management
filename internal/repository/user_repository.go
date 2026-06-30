package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
)

// UserRepository interface cho User
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateStatus(ctx context.Context, userID uint64, status string, emailVerified bool) error
	UpdateLastLogin(ctx context.Context, userID uint64) error
	IncrementFailedLogins(ctx context.Context, userID uint64) (int, error)
	LockAccount(ctx context.Context, userID uint64, lockedUntil time.Time) error
	FindByID(ctx context.Context, userID uint64) (*model.User, error)
	UpdatePassword(ctx context.Context, userID uint64, passwordHash string) error
	UnlockAccount(ctx context.Context, userID uint64) error
}

// userRepository struct
type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository là constructor tạo mới UserRepository
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

// Tạo user mới
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (username, email, password_hash, full_name, phone, date_of_birth, status, email_verified, created_at, updated_at)
		VALUES (:username, :email, :password_hash, :full_name, :phone, :date_of_birth, :status, :email_verified, NOW(), NOW())`
	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		user.ID = uint64(id)
	}
	return err
}

// Tìm user theo email và trả về 1 user
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = ? LIMIT 1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername tìm user theo username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE username = ? LIMIT 1`
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateStatus cập nhật trạng thái và email verified của user
func (r *userRepository) UpdateStatus(ctx context.Context, userID uint64, status string, emailVerified bool) error {
	query := `UPDATE users SET status = ?, email_verified = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, emailVerified, userID)
	return err
}

// UpdateLastLogin cập nhật thời gian đăng nhập, reset failed attempts và mở khóa (nếu có)
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET last_login_at = NOW(), failed_login_attempts = 0, status = 'active', locked_until = NULL, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// IncrementFailedLogins tăng số lần đăng nhập sai một cách nguyên tử và trả về số lần sai mới
func (r *userRepository) IncrementFailedLogins(ctx context.Context, userID uint64) (int, error) {
	updateQuery := "UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at = NOW() WHERE id = ?"
	if _, err := r.db.ExecContext(ctx, updateQuery, userID); err != nil {
		return 0, err
	}

	var attempts int
	err := r.db.GetContext(ctx, &attempts, "SELECT failed_login_attempts FROM users WHERE id = ?", userID)
	return attempts, err
}

// LockAccount khóa tài khoản đến một thời điểm
func (r *userRepository) LockAccount(ctx context.Context, userID uint64, lockedUntil time.Time) error {
	query := `UPDATE users SET status = ?, locked_until = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, model.StatusLocked, lockedUntil, userID)
	return err
}

// FindByID tìm user theo ID
func (r *userRepository) FindByID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = ? LIMIT 1`
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdatePassword cập nhật mật khẩu mới cho user
func (r *userRepository) UpdatePassword(ctx context.Context, userID uint64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, passwordHash, userID)
	return err
}

// UnlockAccount mở khóa tài khoản, reset số lần đăng nhập sai
func (r *userRepository) UnlockAccount(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET status = ?, locked_until = NULL, failed_login_attempts = 0, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, model.StatusActive, userID)
	return err
}


