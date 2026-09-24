package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/bookly-kbtu/backend/internal/domain"
)

const uniqueViolation = "23505"

// Querier is implemented by both *sqlx.DB and *sqlx.Tx,
// allowing repository methods to work inside or outside a transaction.
type Querier interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
}

type DB struct {
	*sqlx.DB
}

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)

	return &DB{db}, nil
}

type txKey struct{}

// WithinTx runs fn in a transaction carried by ctx. Repositories pick it up via Q.
// Nested calls reuse the outer transaction.
func (db *DB) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return fn(ctx)
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err = fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Q returns the transaction from ctx, or the pool when there is none.
func (db *DB) Q(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db.DB
}

// Get scans a single row into dest, returning domain.ErrNotFound on sql.ErrNoRows.
func Get[T any](ctx context.Context, q Querier, dest *T, query string, args ...any) error {
	if err := q.GetContext(ctx, dest, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

// List scans multiple rows into a slice of T.
func List[T any](ctx context.Context, q Querier, query string, args ...any) ([]T, error) {
	var rows []T
	if err := q.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

// Exec maps unique violations to domain.ErrConflict.
func Exec(ctx context.Context, q Querier, query string, args ...any) error {
	_, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.ConstraintName)
		}
		return err
	}
	return nil
}
