package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migrate_mysql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/quocdev03/user-access-management/internal/config"
)

// RunMigrations thực thi các script migration (.sql) để cập nhật schema cơ sở dữ liệu
func RunMigrations(cfg config.DatabaseConfig, migrationsPath string) error {
	// Tạo cấu hình DSN riêng cho migration với MultiStatements = true
	mysqlCfg := mysql.Config{
		User:                 cfg.User,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		DBName:               cfg.Name,
		Collation:            "utf8mb4_unicode_ci",
		ParseTime:            true,
		Loc:                  time.UTC,
		Timeout:              10 * time.Second,
		AllowNativePasswords: true,
		MultiStatements:      true, // Cần thiết để thực thi các script chứa nhiều câu lệnh SQL ngăn cách bởi dấu chấm phẩy
	}

	if cfg.Host != "localhost" && cfg.Host != "mysql" {
		mysqlCfg.TLSConfig = "skip-verify"
	}

	dsn := mysqlCfg.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("không thể kết nối database cho migration: %w", err)
	}
	defer db.Close()

	driver, err := migrate_mysql.WithInstance(db, &migrate_mysql.Config{})
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
