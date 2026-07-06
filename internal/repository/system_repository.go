package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/pkg/database"
)

type AuditLogRepository struct {
	db *sqlx.DB
}

func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			user_id, action, resource, resource_id, ip_address, user_agent, old_values, new_values, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`
	res, err := database.GetDB(ctx, r.db).ExecContext(ctx, query,
		log.UserID,
		log.Action,
		log.Resource,
		log.ResourceID,
		log.IPAddress,
		log.UserAgent,
		log.OldValues,
		log.NewValues,
		log.Status,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err == nil {
		log.ID = uint64(id)
	}

	return nil
}

func (r *AuditLogRepository) FindAllWithFilters(
	ctx context.Context,
	userID *uint64,
	action *string,
	start, end *string,
) ([]model.AuditLog, error) {
	query := `SELECT * FROM audit_logs WHERE 1=1`
	args := []interface{}{}

	if userID != nil {
		query += ` AND user_id = ?`
		args = append(args, *userID)
	}
	if action != nil && *action != "" {
		query += ` AND action = ?`
		args = append(args, *action)
	}
	if start != nil && *start != "" {
		query += ` AND created_at >= ?`
		args = append(args, *start)
	}
	if end != nil && *end != "" {
		query += ` AND created_at <= ?`
		args = append(args, *end)
	}

	query += ` ORDER BY created_at DESC LIMIT 10000`

	logs := []model.AuditLog{}
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &logs, query, args...)
	return logs, err
}
