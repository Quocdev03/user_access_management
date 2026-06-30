package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/pkg/database"
)

type PasswordRepository struct {
	db *sqlx.DB
}

func NewPasswordRepository(db *sqlx.DB) *PasswordRepository {
	return &PasswordRepository{db: db}
}

func (r *PasswordRepository) Create(ctx context.Context, userID uint64, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) 
		VALUES (?, ?, ?)
	`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("PasswordRepository.Create: %w", err)
	}
	return nil
}

func (r *PasswordRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token_hash, is_used, expires_at, created_at 
		FROM password_reset_tokens 
		WHERE token_hash = ?
	`
	var t model.PasswordResetToken
	err := database.GetDB(ctx, r.db).QueryRowContext(ctx, query, tokenHash).Scan(
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
		return nil, fmt.Errorf("PasswordRepository.FindByTokenHash: %w", err)
	}
	return &t, nil
}

func (r *PasswordRepository) MarkAsUsed(ctx context.Context, id uint64) error {
	query := `UPDATE password_reset_tokens SET is_used = 1 WHERE id = ? AND is_used = 0 AND expires_at > NOW()`
	res, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("PasswordRepository.MarkAsUsed: %w", err)
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("PasswordRepository.MarkAsUsed: get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("PasswordRepository.MarkAsUsed: token already used, expired, or not found")
	}
	return nil
}

func (r *PasswordRepository) InvalidateAllUserTokens(ctx context.Context, userID uint64) error {
	query := `UPDATE password_reset_tokens SET is_used = 1 WHERE user_id = ? AND is_used = 0`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("PasswordRepository.InvalidateAllUserTokens: %w", err)
	}
	return nil
}
