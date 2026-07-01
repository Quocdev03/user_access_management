package database

import (
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/config"
)

// ConnectMySQL khởi tạo một connection pool kết nối tới cơ sở dữ liệu MySQL
func ConnectMySQL(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	// Sử dụng mysql.Config để xây dựng DSN chuẩn Clean Code
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
	}

	// Cấu hình bắt buộc cho Render/Aiven
	if cfg.Host != "localhost" && cfg.Host != "mysql" {
		mysqlCfg.TLSConfig = "skip-verify"
	}

	dsn := mysqlCfg.FormatDSN()

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Thiết lập cấu hình connection pool khuyến nghị cho Go
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return db, nil
}
