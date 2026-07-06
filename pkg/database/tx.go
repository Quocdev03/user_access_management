package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type txKey struct{}

func GetDB(ctx context.Context, defaultDB *sqlx.DB) DBExecutor {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return defaultDB
}

type TxManager struct {
	db *sqlx.DB
}

func NewTxManager(db *sqlx.DB) *TxManager {
	return &TxManager{db: db}
}

// RunInTx thực thi hàm fn trong ngữ cảnh của một transaction.
// Nếu context truyền vào đã chứa transaction (Nested transaction), nó sẽ chia sẻ chung kết nối hiện tại.
// Chú ý: TUYỆT ĐỐI không bắt lỗi từ một nested call rồi tiếp tục thao tác trên cùng transaction đó,
// vì kết nối db đằng sau đã bị poison và bắt buộc phải bị rollback toàn bộ bởi transaction cha ở ngoài cùng.
func (tm *TxManager) RunInTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return fn(ctx)
	}

	tx, err := tm.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("không thể bắt đầu transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("lỗi transaction: %v, lỗi rollback: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("lỗi khi commit transaction: %w", err)
	}

	return nil
}
