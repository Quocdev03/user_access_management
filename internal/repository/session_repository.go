package repository

import (
	"context"
	"database/sql"
	"errors"
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
		INSERT INTO sessions (user_id, token_hash, refresh_token_hash, ip_address, user_agent, device_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
	`
	res, err := database.GetDB(ctx, r.db).ExecContext(ctx, query,
		session.UserID,
		session.TokenHash,
		session.RefreshTokenHash,
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
		SET token_hash = ?, refresh_token_hash = ?, ip_address = ?, user_agent = ?, device_id = ?, expires_at = ?
		WHERE id = ?
	`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query,
		session.TokenHash,
		session.RefreshTokenHash,
		session.IPAddress,
		session.UserAgent,
		session.DeviceID,
		session.ExpiresAt,
		session.ID,
	)
	return err
}

func (r *SessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*model.Session, error) {
	var session model.Session
	query := `SELECT * FROM sessions WHERE refresh_token_hash = ? LIMIT 1`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &session, query, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
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

func (r *SessionRepository) SetRateLimit(ctx context.Context, action string, identifier string, ttl time.Duration) error {
	key := "ratelimit:" + action + ":" + identifier
	return r.redis.Set(ctx, key, "1", ttl).Err()
}

func (r *SessionRepository) IsRateLimited(ctx context.Context, action string, identifier string) (bool, error) {
	key := "ratelimit:" + action + ":" + identifier
	val, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
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
