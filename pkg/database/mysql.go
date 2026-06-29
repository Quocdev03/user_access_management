package database

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/quocdev03/user-access-management/internal/config"
)

// ConnectMySQL khởi tạo một connection pool kết nối tới cơ sở dữ liệu MySQL
func ConnectMySQL(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	// Định dạng DSN: username:password@protocol(address)/dbname?param=value
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC&timeout=10s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

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
