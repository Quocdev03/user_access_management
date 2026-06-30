package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// DBExecutor định nghĩa các method chung cho cả *sqlx.DB và *sqlx.Tx
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type txKey struct{}

// GetDB lấy ra DBExecutor từ context.
// Nếu trong context có sẵn Transaction (do RunInTx truyền vào), nó sẽ trả về Tx.
// Ngược lại, nó sẽ trả về db mặc định (*sqlx.DB).
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

// RunInTx nhận vào một function và chạy nó bên trong một Database Transaction.
// Nếu hàm trả về error, giao dịch sẽ bị Rollback. Nếu thành công, giao dịch sẽ Commit.
func (tm *TxManager) RunInTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	// Nếu context đã nằm trong 1 Tx rồi, ta chỉ việc chạy tiếp (không tạo nested tx).
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
