package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenant runs fn in a transaction in which every query sees only the
// rows of the pharmacy pharmacyID (migration 00003). If fn returns an error,
// the transaction is rolled back and WithTenant returns that error as is;
// otherwise it commits.
//
// The tenant is set with set_config(…, true), so it ends with the
// transaction: the pooled connection goes back with no tenant on it, and
// the next query on it without WithTenant fails.
//
// Pass fn's tx to queries.New to use the generated queries inside it.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, pharmacyID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning the transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.pharmacy_id', $1, true)", pharmacyID.String()); err != nil {
		return fmt.Errorf("setting the tenant: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	return nil
}
