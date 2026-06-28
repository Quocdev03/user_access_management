package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
)

// RoleRepository interface cho Role
type RoleRepository interface {
	FindByName(ctx context.Context, name string) (*model.Role, error)
	AssignRoleToUser(ctx context.Context, userID, roleID uint64) error
}

type roleRepository struct {
	db *sqlx.DB
}

// NewRoleRepository là constructor tạo mới RoleRepository
func NewRoleRepository(db *sqlx.DB) RoleRepository {
	return &roleRepository{db: db}
}

// FindByName tìm role theo tên
func (r *roleRepository) FindByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	query := `SELECT * FROM roles WHERE name = ? LIMIT 1`
	err := r.db.GetContext(ctx, &role, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// AssignRoleToUser gán role cho user
func (r *roleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uint64) error {
	query := `INSERT INTO user_roles (user_id, role_id, assigned_at) VALUES (?, ?, NOW())`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	return err
}
