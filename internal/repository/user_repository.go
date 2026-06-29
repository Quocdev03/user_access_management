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
	IncrementFailedLogins(ctx context.Context, userID uint64) error
	LockAccount(ctx context.Context, userID uint64, lockedUntil time.Time) error
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

// UpdateLastLogin cập nhật thời gian đăng nhập và reset failed attempts
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET last_login_at = NOW(), failed_login_attempts = 0, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// IncrementFailedLogins tăng số lần đăng nhập sai
func (r *userRepository) IncrementFailedLogins(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// LockAccount khóa tài khoản đến một thời điểm
func (r *userRepository) LockAccount(ctx context.Context, userID uint64, lockedUntil time.Time) error {
	query := `UPDATE users SET status = ?, locked_until = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, model.StatusLocked, lockedUntil, userID)
	return err
}
