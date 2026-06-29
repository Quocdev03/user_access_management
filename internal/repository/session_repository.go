package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/redis/go-redis/v9"
)

// SessionRepository định nghĩa các giao tiếp dữ liệu liên quan đến phiên làm việc của người dùng
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	Update(ctx context.Context, session *model.Session) error
	FindByRefreshTokenHash(ctx context.Context, hash string) (*model.Session, error)
	DeleteByRefreshTokenHash(ctx context.Context, hash string) error
	DeleteByTokenHash(ctx context.Context, hash string) error
	DeleteByUserID(ctx context.Context, userID uint64) error
	AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

type sessionRepository struct {
	db    *sqlx.DB
	redis *redis.Client
}

// NewSessionRepository khởi tạo đối tượng sessionRepository thực thi SessionRepository interface
func NewSessionRepository(db *sqlx.DB, redis *redis.Client) SessionRepository {
	return &sessionRepository{
		db:    db,
		redis: redis,
	}
}

// Create lưu thông tin phiên đăng nhập (session) mới của người dùng vào MySQL
func (r *sessionRepository) Create(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (user_id, token_hash, refresh_token_hash, ip_address, user_agent, device_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
	`
	res, err := r.db.ExecContext(ctx, query,
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

// Update cập nhật thông tin session hiện tại (dùng khi xoay vòng Refresh Token/UPDATE session)
func (r *sessionRepository) Update(ctx context.Context, session *model.Session) error {
	query := `
		UPDATE sessions 
		SET token_hash = ?, refresh_token_hash = ?, ip_address = ?, user_agent = ?, device_id = ?, expires_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
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

// FindByRefreshTokenHash tìm kiếm một session hợp lệ bằng mã băm refresh token
func (r *sessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*model.Session, error) {
	var session model.Session
	query := `SELECT * FROM sessions WHERE refresh_token_hash = ? LIMIT 1`
	err := r.db.GetContext(ctx, &session, query, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// DeleteByRefreshTokenHash xóa bản ghi session khỏi MySQL dựa trên refresh token hash
func (r *sessionRepository) DeleteByRefreshTokenHash(ctx context.Context, hash string) error {
	query := `DELETE FROM sessions WHERE refresh_token_hash = ?`
	_, err := r.db.ExecContext(ctx, query, hash)
	return err
}

// DeleteByTokenHash xóa bản ghi session dựa trên token hash (thường dùng khi đăng xuất)
func (r *sessionRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	query := `DELETE FROM sessions WHERE token_hash = ?`
	_, err := r.db.ExecContext(ctx, query, hash)
	return err
}

// DeleteByUserID thu hồi toàn bộ các phiên hoạt động của một người dùng nhất định
func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uint64) error {
	query := `DELETE FROM sessions WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// AddToBlacklist thêm mã định danh JTI của Access Token bị thu hồi vào Redis blacklist
func (r *sessionRepository) AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error {
	return r.redis.Set(ctx, "blacklist:"+jti, "1", ttl).Err()
}

// IsBlacklisted kiểm tra xem Access Token (JTI) có nằm trong danh sách đen bị thu hồi hay không
func (r *sessionRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	val, err := r.redis.Exists(ctx, "blacklist:"+jti).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}
