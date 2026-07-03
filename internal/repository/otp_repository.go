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
	now := time.Now().UTC()

	if forUpdate {
		var id uint64
		idQuery := `
			SELECT id FROM otp_codes 
			WHERE user_id = ? AND type = ? AND is_used = false AND expires_at > ? 
			ORDER BY created_at DESC LIMIT 1
		`
		err := database.GetDB(ctx, r.db).GetContext(ctx, &id, idQuery, userID, otpType, now)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}

		err = database.GetDB(ctx, r.db).GetContext(ctx, &otp, "SELECT * FROM otp_codes WHERE id = ? AND is_used = false FOR UPDATE", id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		return &otp, nil
	}

	query := `
		SELECT * FROM otp_codes 
		WHERE user_id = ? AND type = ? AND is_used = false AND expires_at > ? 
		ORDER BY created_at DESC LIMIT 1
	`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &otp, query, userID, otpType, now)
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
	var attempts int
	selectQuery := "SELECT attempts FROM otp_codes WHERE id = ? FOR UPDATE"
	if err := database.GetDB(ctx, r.db).GetContext(ctx, &attempts, selectQuery, otpID); err != nil {
		return 0, err
	}

	attempts++
	updateQuery := "UPDATE otp_codes SET attempts = ? WHERE id = ?"
	if _, err := database.GetDB(ctx, r.db).ExecContext(ctx, updateQuery, attempts, otpID); err != nil {
		return 0, err
	}

	return attempts, nil
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
