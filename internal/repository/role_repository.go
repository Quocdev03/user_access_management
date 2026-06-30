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
