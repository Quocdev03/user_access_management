package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/pkg/database"
)

type OTPRepository struct {
	db *sqlx.DB
}

func NewOTPRepository(db *sqlx.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

func (r *OTPRepository) Create(ctx context.Context, userID uint64, code string, otpType string, expiresAt time.Time) error {
	queryDisable := `UPDATE otp_codes SET is_used = true WHERE user_id = ? AND type = ? AND is_used = false`
	if _, err := database.GetDB(ctx, r.db).ExecContext(ctx, queryDisable, userID, otpType); err != nil {
		return err
	}

	query := `
		INSERT INTO otp_codes (user_id, code, type, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, userID, code, otpType, expiresAt, time.Now().UTC())
	return err
}

func (r *OTPRepository) getLatestValidCode(ctx context.Context, userID uint64, otpType string, forUpdate bool) (*model.OTPCode, error) {
	var otp model.OTPCode
	query := `
		SELECT * FROM otp_codes 
		WHERE user_id = ? AND type = ? AND is_used = false AND expires_at > ? 
		ORDER BY created_at DESC LIMIT 1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}
	err := database.GetDB(ctx, r.db).GetContext(ctx, &otp, query, userID, otpType, time.Now().UTC())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &otp, nil
}

func (r *OTPRepository) GetLatestValidCode(ctx context.Context, userID uint64, otpType string) (*model.OTPCode, error) {
	return r.getLatestValidCode(ctx, userID, otpType, false)
}

func (r *OTPRepository) GetLatestValidCodeForUpdate(ctx context.Context, userID uint64, otpType string) (*model.OTPCode, error) {
	return r.getLatestValidCode(ctx, userID, otpType, true)
}

func (r *OTPRepository) IncrementAttempts(ctx context.Context, otpID uint64) (int, error) {
	updateQuery := "UPDATE otp_codes SET attempts = attempts + 1 WHERE id = ?"
	if _, err := database.GetDB(ctx, r.db).ExecContext(ctx, updateQuery, otpID); err != nil {
		return 0, err
	}

	var attempts int
	err := database.GetDB(ctx, r.db).GetContext(ctx, &attempts, "SELECT attempts FROM otp_codes WHERE id = ?", otpID)
	return attempts, err
}

func (r *OTPRepository) MarkAsUsed(ctx context.Context, otpID uint64) error {
	query := `UPDATE otp_codes SET is_used = true WHERE id = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, otpID)
	return err
}

func (r *OTPRepository) DeleteExpired(ctx context.Context, threshold time.Time) error {
	query := `DELETE FROM otp_codes WHERE expires_at < ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, threshold)
	return err
}
