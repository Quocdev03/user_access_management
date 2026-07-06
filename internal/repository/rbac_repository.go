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
	roles := []string{}
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
	permission := []string{}
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
	rows := []userRole{}

	if err := database.GetDB(ctx, r.db).SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row.Name)
	}

	return result, nil
}

func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	query := `INSERT INTO roles (name, description, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`
	res, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, role.Name, role.Description)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	role.ID = uint64(id)
	return nil
}

func (r *RoleRepository) Update(ctx context.Context, role *model.Role) error {
	query := `UPDATE roles SET name = ?, description = ? WHERE id = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, role.Name, role.Description, role.ID)
	return err
}

func (r *RoleRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM roles WHERE id = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, id)
	return err
}

func (r *RoleRepository) FindAll(ctx context.Context) ([]model.Role, error) {
	roles := []model.Role{}
	query := `SELECT * FROM roles ORDER BY id ASC`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &roles, query)
	return roles, err
}

func (r *RoleRepository) GetPermissionsByRoleID(ctx context.Context, roleID uint64) ([]model.Permission, error) {
	perms := []model.Permission{}
	query := `
		SELECT p.* FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = ?
	`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &perms, query, roleID)
	return perms, err
}

func (r *RoleRepository) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = ?`, roleID)
	if err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	query := `INSERT INTO role_permissions (role_id, permission_id, assigned_at) VALUES `
	args := []interface{}{}
	for i, pid := range permissionIDs {
		if i > 0 {
			query += `, `
		}
		query += `(?, ?, NOW())`
		args = append(args, roleID, pid)
	}
	_, err = database.GetDB(ctx, r.db).ExecContext(ctx, query, args...)
	return err
}

func (r *RoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uint64) error {
	query := `DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`
	_, err := database.GetDB(ctx, r.db).ExecContext(ctx, query, userID, roleID)
	return err
}


type PermissionRepository struct {
	db *sqlx.DB
}

func NewPermissionRepository(db *sqlx.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) FindAll(ctx context.Context) ([]model.Permission, error) {
	perms := []model.Permission{}
	query := `SELECT * FROM permissions ORDER BY id ASC`
	err := database.GetDB(ctx, r.db).SelectContext(ctx, &perms, query)
	return perms, err
}
