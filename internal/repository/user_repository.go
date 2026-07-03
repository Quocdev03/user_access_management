package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/constant"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/pkg/database"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	query := `INSERT INTO users (username, email, password_hash, full_name, phone, date_of_birth, status, email_verified, created_at, updated_at)
		VALUES (:username, :email, :password_hash, :full_name, :phone, :date_of_birth, :status, :email_verified, :created_at, :updated_at)`
	result, err := database.GetDB(ctx, r.db).NamedExecContext(ctx, query, user)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		user.ID = uint64(id)
	}
	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = ? LIMIT 1`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}



func (r *UserRepository) FindByID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = ? LIMIT 1`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmailForUpdate(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = ? LIMIT 1 FOR UPDATE`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByIDForUpdate(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = ? LIMIT 1 FOR UPDATE`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser cập nhật toàn bộ thông tin của user.
// CẢNH BÁO: Để tránh rủi ro Lost Update trong môi trường concurrent,
// caller nên chạy hàm này trong một transaction và fetch user
// bằng SELECT FOR UPDATE trước khi thực hiện cập nhật.
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()
	query := `UPDATE users SET 
		username = :username,
		email = :email,
		password_hash = :password_hash,
		full_name = :full_name,
		phone = :phone,
		avatar_url = :avatar_url,
		date_of_birth = :date_of_birth,
		status = :status,
		email_verified = :email_verified,
		failed_login_attempts = :failed_login_attempts,
		locked_until = :locked_until,
		last_login_at = :last_login_at,
		updated_at = :updated_at
		WHERE id = :id`
	_, err := database.GetDB(ctx, r.db).NamedExecContext(ctx, query, user)
	return err
}

// IncrementFailedLogins tăng số lần đăng nhập sai một cách nguyên tử (dùng optimistic locking)
func (r *UserRepository) IncrementFailedLogins(ctx context.Context, userID uint64) (int, error) {
	for i := 0; i < 3; i++ {
		var attempts int
		if err := database.GetDB(ctx, r.db).GetContext(ctx, &attempts, "SELECT failed_login_attempts FROM users WHERE id = ?", userID); err != nil {
			return 0, err
		}

		newAttempts := attempts + 1
		updateQuery := "UPDATE users SET failed_login_attempts = ?, updated_at = ? WHERE id = ? AND failed_login_attempts = ?"
		res, err := database.GetDB(ctx, r.db).ExecContext(ctx, updateQuery, newAttempts, time.Now().UTC(), userID, attempts)
		if err != nil {
			return 0, err
		}

		rows, _ := res.RowsAffected()
		if rows == 1 {
			return newAttempts, nil
		}
	}
	return 0, errors.New("không thể tăng số lần đăng nhập sai do xung đột dữ liệu")
}

// LockAccount khóa tài khoản một cách nguyên tử để tránh race condition
func (r *UserRepository) LockAccount(ctx context.Context, userID uint64, lockedUntil time.Time, attempts int) error {
	query := "UPDATE users SET status = ?, locked_until = ?, failed_login_attempts = ?, updated_at = ? WHERE id = ?"
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, constant.StatusLocked, lockedUntil, attempts, time.Now().UTC(), userID)
	return err
}

// UnlockIfExpired mở khóa tài khoản nếu thời gian khóa đã qua, dùng atomic query để chống Lost Update
func (r *UserRepository) UnlockIfExpired(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET status = ?, locked_until = NULL, failed_login_attempts = 0, updated_at = ? 
              WHERE id = ? AND status = ? AND locked_until <= ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, constant.StatusActive, time.Now().UTC(), userID, constant.StatusLocked, time.Now().UTC())
	return err
}

