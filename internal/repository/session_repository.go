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

func (r *SessionRepository) IncrementRateLimit(ctx context.Context, action string, identifier string, limit int, window time.Duration) (bool, error) {
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
