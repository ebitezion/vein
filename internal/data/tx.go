package data

import (
	"context"
	"database/sql"
	"time"
)

type TxManager struct {
	DB *sql.DB
}

// WithTransaction executes fn in a single database transaction.
func (tm TxManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tx, err := tm.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// NewTimeoutContext provides a standard DB timeout helper for repository usage.
func NewTimeoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, 3*time.Second)
}
