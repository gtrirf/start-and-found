// Package database owns the PostgreSQL connection pool and the transaction
// helpers used by the domain repositories.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQL error codes used to translate storage failures into API errors.
const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
	checkViolationCode      = "23514"
)

// Querier is the subset of pgx used by repositories. Both *pgxpool.Pool and
// pgx.Tx satisfy it, so a service can transparently run a repository inside a
// transaction by building it with the transactional querier.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB owns the connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Connect opens the pool and waits for the database to accept connections,
// which keeps container start-up order irrelevant during development.
func Connect(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		pingErr := pool.Ping(pingCtx)
		cancel()
		if pingErr == nil {
			break
		}
		if time.Now().After(deadline) {
			pool.Close()
			return nil, fmt.Errorf("connect to postgres: %w", pingErr)
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return &DB{pool: pool}, nil
}

// Pool exposes the underlying pool for read only access.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Ping verifies that the database is reachable.
func (db *DB) Ping(ctx context.Context) error { return db.pool.Ping(ctx) }

// Close releases every pooled connection.
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// WithTx runs fn inside a transaction, rolling back on error or panic.
func (db *DB) WithTx(ctx context.Context, fn func(q Querier) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			_ = tx.Rollback(rollbackCtx)
			cancel()
			panic(recovered)
		}
	}()

	if err := fn(tx); err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		_ = tx.Rollback(rollbackCtx)
		cancel()
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// IsUniqueViolation reports whether err violates a unique constraint.
func IsUniqueViolation(err error) bool { return pgErrorCode(err) == uniqueViolationCode }

// IsForeignKeyViolation reports whether err violates a foreign key constraint.
func IsForeignKeyViolation(err error) bool { return pgErrorCode(err) == foreignKeyViolationCode }

// IsCheckViolation reports whether err violates a CHECK constraint.
func IsCheckViolation(err error) bool { return pgErrorCode(err) == checkViolationCode }

// ConstraintName returns the constraint reported by PostgreSQL, if any.
func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
