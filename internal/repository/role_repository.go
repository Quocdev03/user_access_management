package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/model"
	"github.com/quocdev03/user-access-management/pkg/database"
)

type RoleRepository struct {
	db *sqlx.DB
}

func NewRoleRepository(db *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) FindByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	query := `SELECT * FROM roles WHERE name = ? LIMIT 1`
	err := database.GetDB(ctx, r.db).GetContext(ctx, &role, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uint64) error {
	query := `INSERT INTO user_roles (user_id, role_id, assigned_at) VALUES (?, ?, NOW())`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, userID, roleID)
	return err
}

func (r *RoleRepository) GetRolesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	var roles []string
	query := `
		SELECT r.name 
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = ?
	`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepository) GetPermissionsByUserId(ctx context.Context, userID uint64) ([]string, error) {
	var permission []string
	query := `
		SELECT DISTINCT p.name 
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?
	`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &permission, query, userID)
	return permission, err
}

func (r *RoleRepository) GetRolesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64][]string, error) {
	result := make(map[uint64][]string)
	if len(userIDs) == 0 {
		return result, nil
	}

	query, args, err := sqlx.In(`
		SELECT ur.user_id, r.name 
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id IN (?)
	`, userIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	type userRole struct {
		UserID uint64 `db:"user_id"`
		Name   string `db:"name"`
	}
	var rows []userRole

	if err := database.GetDB(ctx, r.db).SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row.Name)
	}

	return result, nil
}
