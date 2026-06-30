package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
)

// OTPRepository interface cho OTP
type OTPRepository interface {
	Create(ctx context.Context, userID uint64, code string, otpType string, expiresAt time.Time) error
	GetLatestValidCode(ctx context.Context, userID uint64, otpType string) (*model.OTPCode, error)
	IncrementAttempts(ctx context.Context, otpID uint64) (int, error)
	MarkAsUsed(ctx context.Context, otpID uint64) error
}

// otpRepository struct
type otpRepository struct {
	db *sqlx.DB
}

// NewOTPRepository là constructor tạo mới OTPRepository
func NewOTPRepository(db *sqlx.DB) OTPRepository {
	return &otpRepository{db: db}
}

// Create tạo OTP mới (đánh dấu cũ là used nếu có)
func (r *otpRepository) Create(ctx context.Context, userID uint64, code string, otpType string, expiresAt time.Time) error {
	// Ẩn OTP cũ cùng loại
	queryDisable := `UPDATE otp_codes SET is_used = true WHERE user_id = ? AND type = ? AND is_used = false`
	_, _ = r.db.ExecContext(ctx, queryDisable, userID, otpType)

	query := `
		INSERT INTO otp_codes (user_id, code, type, expires_at, created_at)
		VALUES (?, ?, ?, ?, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, userID, code, otpType, expiresAt)
	return err
}

// GetLatestValidCode lấy thông tin mã OTP hợp lệ mới nhất
func (r *otpRepository) GetLatestValidCode(ctx context.Context, userID uint64, otpType string) (*model.OTPCode, error) {
	var otp model.OTPCode
	query := `
		SELECT * FROM otp_codes 
		WHERE user_id = ? AND type = ? AND is_used = false AND expires_at > NOW() 
		ORDER BY created_at DESC LIMIT 1
	`
	err := r.db.GetContext(ctx, &otp, query, userID, otpType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Không có OTP hoặc đã hết hạn/sử dụng
		}
		return nil, err
	}
	return &otp, nil
}

// IncrementAttempts tăng số lần thử sai một cách nguyên tử và trả về số lần hiện tại
func (r *otpRepository) IncrementAttempts(ctx context.Context, otpID uint64) (int, error) {
	updateQuery := "UPDATE otp_codes SET attempts = attempts + 1 WHERE id = ?"
	if _, err := r.db.ExecContext(ctx, updateQuery, otpID); err != nil {
		return 0, err
	}

	var attempts int
	err := r.db.GetContext(ctx, &attempts, "SELECT attempts FROM otp_codes WHERE id = ?", otpID)
	return attempts, err
}

// MarkAsUsed đánh dấu đã sử dụng
func (r *otpRepository) MarkAsUsed(ctx context.Context, otpID uint64) error {
	query := `UPDATE otp_codes SET is_used = true WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, otpID)
	return err
}
