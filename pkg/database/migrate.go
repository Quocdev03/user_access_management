package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations thực thi các script migration (.sql) để cập nhật schema cơ sở dữ liệu
func RunMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("không thể khởi tạo migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("không thể khởi tạo migrate instance: %w", err)
	}

	// Cơ chế tự động Auto-Heal (cứu hộ tự động) không code cứng version:
	// Đọc phiên bản hiện tại, nếu phát hiện database đang bị "Dirty" (kẹt do lỗi đứt ngang),
	// thì tự động Force lùi về 1 phiên bản trước đó để chạy lại từ đầu file bị lỗi.
	version, dirty, err := m.Version()
	if err == nil && dirty {
		if forceErr := m.Force(int(version) - 1); forceErr != nil {
			return fmt.Errorf("lỗi khi tự động force dirty database: %w", forceErr)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("lỗi khi chạy migration: %w", err)
	}

	return nil
}
