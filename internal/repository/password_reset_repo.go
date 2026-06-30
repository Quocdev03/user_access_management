package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/quocdev03/user-access-management/internal/model"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, userID uint64, tokenHash string, expiresAt time.Time) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, id uint64) error
	InvalidateAllUserTokens(ctx context.Context, userID uint64) error
}

type passwordResetRepo struct {
	db *sql.DB
}

func NewPasswordResetRepository(db *sql.DB) PasswordResetRepository {
	return &passwordResetRepo{db: db}
}

func (r *passwordResetRepo) Create(ctx context.Context, userID uint64, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) 
		VALUES (?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("PasswordResetRepository.Create: %w", err)
	}
	return nil
}

func (r *passwordResetRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token_hash, is_used, expires_at, created_at 
		FROM password_reset_tokens 
		WHERE token_hash = ?
	`
	var t model.PasswordResetToken
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.IsUsed,
		&t.ExpiresAt,
		&t.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("PasswordResetRepository.FindByTokenHash: %w", err)
	}
	return &t, nil
}

func (r *passwordResetRepo) MarkAsUsed(ctx context.Context, id uint64) error {
	query := `UPDATE password_reset_tokens SET is_used = 1 WHERE id = ? AND is_used = 0 AND expires_at > NOW()`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("PasswordResetRepository.MarkAsUsed: %w", err)
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("PasswordResetRepository.MarkAsUsed: get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("PasswordResetRepository.MarkAsUsed: token already used, expired, or not found")
	}
	return nil
}

func (r *passwordResetRepo) InvalidateAllUserTokens(ctx context.Context, userID uint64) error {
	query := `UPDATE password_reset_tokens SET is_used = 1 WHERE user_id = ? AND is_used = 0`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("PasswordResetRepository.InvalidateAllUserTokens: %w", err)
	}
	return nil
}
