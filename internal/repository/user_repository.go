package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
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
	query := `INSERT INTO users (username, email, password_hash, full_name, phone, date_of_birth, status, email_verified, created_at, updated_at)
		VALUES (:username, :email, :password_hash, :full_name, :phone, :date_of_birth, :status, :email_verified, NOW(), NOW())`
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

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE username = ? LIMIT 1`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &user, query, username)
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

func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
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
		updated_at = NOW()
		WHERE id = :id`
	_, err := database.GetDB(ctx, r.db).NamedExecContext(ctx, query, user)
	return err
}

// IncrementFailedLogins tăng số lần đăng nhập sai một cách nguyên tử
func (r *UserRepository) IncrementFailedLogins(ctx context.Context, userID uint64) (int, error) {
	updateQuery := "UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at = NOW() WHERE id = ?"
	if _, err := database.GetDB(ctx, r.db).ExecContext(ctx, updateQuery, userID); err != nil {
		return 0, err
	}

	var attempts int
	err := database.GetDB(ctx, r.db).GetContext(ctx, &attempts, "SELECT failed_login_attempts FROM users WHERE id = ?", userID)
	return attempts, err
}
