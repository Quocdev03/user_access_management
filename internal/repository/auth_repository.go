package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/pkg/database"
	"github.com/redis/go-redis/v9"
)

type SessionRepository struct {
	db    *sqlx.DB
	redis *redis.Client
}

func NewSessionRepository(db *sqlx.DB, redis *redis.Client) *SessionRepository {
	return &SessionRepository{
		db:    db,
		redis: redis,
	}
}

func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (user_id, token_hash, refresh_token_hash, jti, ip_address, user_agent, device_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`
	res, err := database.GetDB(ctx, r.db).ExecContext(ctx, query,
		session.UserID,
		session.TokenHash,
		session.RefreshTokenHash,
		session.JTI,
		session.IPAddress,
		session.UserAgent,
		session.DeviceID,
		session.ExpiresAt,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		session.ID = uint64(id)
	}
	return nil
}

func (r *SessionRepository) Update(ctx context.Context, session *model.Session) error {
	query := `
		UPDATE sessions 
		SET token_hash = ?, refresh_token_hash = ?, jti = ?, ip_address = ?, user_agent = ?, device_id = ?, expires_at = ?
		WHERE id = ?
	`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query,
		session.TokenHash,
		session.RefreshTokenHash,
		session.JTI,
		session.IPAddress,
		session.UserAgent,
		session.DeviceID,
		session.ExpiresAt,
		session.ID,
	)
	return err
}

func (r *SessionRepository) findByRefreshTokenHash(ctx context.Context, hash string, forUpdate bool) (*model.Session, error) {
	var session model.Session
	query := `SELECT * FROM sessions WHERE refresh_token_hash = ? LIMIT 1`
	if forUpdate {
		query += " FOR UPDATE"
	}
	err := database.GetDB(ctx, r.db).GetContext(ctx, &session, query, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) FindByRefreshTokenHashForUpdate(ctx context.Context, hash string) (*model.Session, error) {
	return r.findByRefreshTokenHash(ctx, hash, true)
}

func (r *SessionRepository) DeleteByRefreshTokenHash(ctx context.Context, hash string) error {
	query := `DELETE FROM sessions WHERE refresh_token_hash = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, hash)
	return err
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	query := `DELETE FROM sessions WHERE token_hash = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, hash)
	return err
}

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID uint64) error {
	query := `DELETE FROM sessions WHERE user_id = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, userID)
	return err
}

// InvalidateUserSessions xóa mọi session MySQL của user và set Redis revoke epoch
// (access JWT phát hành trước mốc epoch sẽ bị từ chối). Gọi ngoài MySQL transaction.
func (r *SessionRepository) InvalidateUserSessions(ctx context.Context, userID uint64, ttl time.Duration) error {
	if err := r.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("DeleteByUserID: %w", err)
	}
	if err := r.RevokeAllUserTokens(ctx, userID, ttl); err != nil {
		return fmt.Errorf("RevokeAllUserTokens: %w", err)
	}
	return nil
}

func (r *SessionRepository) AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error {
	return r.redis.Set(ctx, "blacklist:"+jti, "1", ttl).Err()
}

func (r *SessionRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	val, err := r.redis.Exists(ctx, "blacklist:"+jti).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (r *SessionRepository) AddRevokedRefreshToken(ctx context.Context, hash string, ttl time.Duration) error {
	return r.redis.Set(ctx, "revoked_rt:"+hash, "1", ttl).Err()
}

func (r *SessionRepository) IsRefreshTokenRevoked(ctx context.Context, hash string) (bool, error) {
	val, err := r.redis.Exists(ctx, "revoked_rt:"+hash).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (r *SessionRepository) IncrementRateLimit(
	ctx context.Context,
	action string,
	identifier string,
	limit int,
	window time.Duration,
) (bool, error) {
	key := "ratelimit:" + action + ":" + identifier

	script := redis.NewScript(`
		local current = redis.call("INCR", KEYS[1])
		if current == 1 then
			redis.call("PEXPIRE", KEYS[1], ARGV[1])
		end
		return current
	`)

	val, err := script.Run(ctx, r.redis, []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}

	return val > int64(limit), nil
}

func (r *SessionRepository) RevokeAllUserTokens(ctx context.Context, userID uint64, ttl time.Duration) error {
	key := "user_revoked_epoch:" + strconv.FormatUint(userID, 10)
	now := time.Now().Unix()
	return r.redis.Set(ctx, key, now, ttl).Err()
}

func (r *SessionRepository) GetUserRevokedEpoch(ctx context.Context, userID uint64) (int64, error) {
	key := "user_revoked_epoch:" + strconv.FormatUint(userID, 10)
	val, err := r.redis.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	return val, nil
}

func (r *SessionRepository) SetEmailChangePending(ctx context.Context, userID uint64, newEmail string, ttl time.Duration) error {
	key := "email_change_pending:" + strconv.FormatUint(userID, 10)
	return r.redis.Set(ctx, key, newEmail, ttl).Err()
}

func (r *SessionRepository) GetEmailChangePending(ctx context.Context, userID uint64) (string, error) {
	key := "email_change_pending:" + strconv.FormatUint(userID, 10)
	val, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

func (r *SessionRepository) DeleteEmailChangePending(ctx context.Context, userID uint64) error {
	keyPending := "email_change_pending:" + strconv.FormatUint(userID, 10)
	return r.redis.Del(ctx, keyPending).Err()
}

func (r *SessionRepository) DeleteExpired(ctx context.Context, threshold time.Time) error {
	query := `DELETE FROM sessions WHERE expires_at < ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, threshold)
	return err
}

func (r *SessionRepository) FindActiveSessionsByUserID(ctx context.Context, userID uint64) ([]model.Session, error) {
	sessions := []model.Session{}
	query := `SELECT * FROM sessions WHERE user_id = ? AND expires_at > NOW() ORDER BY created_at DESC`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &sessions, query, userID)
	return sessions, err
}

func (r *SessionRepository) FindByIDAndUserID(ctx context.Context, id, userID uint64) (*model.Session, error) {
	var session model.Session
	query := `SELECT * FROM sessions WHERE id = ? AND user_id = ? LIMIT 1`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &session, query, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) DeleteByIDAndUserID(ctx context.Context, id, userID uint64) error {
	query := `DELETE FROM sessions WHERE id = ? AND user_id = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, id, userID)
	return err
}

type DeviceRepository struct {
	db *sqlx.DB
}

func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) FindOrCreate(ctx context.Context, d *model.Device) error {
	// Schema: device_name 255 (full UA), os/browser 50, ip 45
	clampDeviceFields(d)

	osVal, browserVal := "", ""
	if d.OS != nil {
		osVal = *d.OS
	}
	if d.Browser != nil {
		browserVal = *d.Browser
	}

	queryFind := `SELECT * FROM devices WHERE user_id = ? AND os = ? AND browser = ? LIMIT 1`
	var existing model.Device
	err := database.GetDB(ctx, r.db).GetContext(ctx, &existing, queryFind, d.UserID, osVal, browserVal)
	if err == nil {
		d.ID = existing.ID
		queryUpdate := `UPDATE devices SET last_active_at = NOW(), ip_address = ?, device_name = COALESCE(?, device_name) WHERE id = ?`
		_, _ = database.GetDB(ctx, r.db).ExecContext(ctx, queryUpdate, d.IPAddress, d.DeviceName, d.ID)
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	queryInsert := `
		INSERT INTO devices (user_id, device_name, device_type, os, browser, ip_address, last_active_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
	`
	res, err := database.GetDB(ctx, r.db).ExecContext(ctx, queryInsert,
		d.UserID, d.DeviceName, d.DeviceType, d.OS, d.Browser, d.IPAddress)
	if err != nil {
		return fmt.Errorf("DeviceRepository.FindOrCreate insert: %w", err)
	}
	id, _ := res.LastInsertId()
	d.ID = uint64(id)
	return nil
}

func clampDeviceFields(d *model.Device) {
	if d.DeviceName != nil {
		s := clampRunes(*d.DeviceName, 255)
		d.DeviceName = &s
	}
	if d.DeviceType != nil {
		s := clampRunes(*d.DeviceType, 50)
		d.DeviceType = &s
	}
	if d.OS != nil {
		s := clampRunes(*d.OS, 50)
		d.OS = &s
	}
	if d.Browser != nil {
		s := clampRunes(*d.Browser, 50)
		d.Browser = &s
	}
	if d.IPAddress != nil {
		s := clampRunes(*d.IPAddress, 45)
		d.IPAddress = &s
	}
}

func clampRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func (r *DeviceRepository) FindByUserID(ctx context.Context, userID uint64) ([]model.Device, error) {
	devices := []model.Device{}
	query := `SELECT * FROM devices WHERE user_id = ? ORDER BY last_active_at DESC`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &devices, query, userID)
	return devices, err
}

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
	query := `
		SELECT * FROM otp_codes 
		WHERE user_id = ? AND type = ? AND is_used = false AND expires_at > ? 
		ORDER BY created_at DESC LIMIT 1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	err := database.GetDB(ctx, r.db).GetContext(ctx, &otp, query, userID, otpType, now)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &otp, nil
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
	selectQuery := "SELECT attempts FROM otp_codes WHERE id = ?"
	if err := database.GetDB(ctx, r.db).GetContext(ctx, &attempts, selectQuery, otpID); err != nil {
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

func (r *PasswordRepository) DeleteExpired(ctx context.Context, threshold time.Time) error {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, threshold)
	return err
}
