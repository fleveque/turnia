package db

import (
	"context"
	"errors"

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
	// YOUR TURN (milestone 3.3): the body, in five steps. inTenant, in
	// tenancy_test.go, does the first three for a test; here, errors are
	// returned instead of failing a test, and a success is committed.
	//
	//  1. Begin a transaction: pool.Begin(ctx) returns a pgx.Tx and an error.
	//     On an error, return it wrapped: fmt.Errorf("beginning the
	//     transaction: %w", err).
	//  2. defer tx.Rollback(ctx). It runs when WithTenant returns, however it
	//     returns. After a commit it does nothing; otherwise it undoes the
	//     transaction and gives its connection back to the pool.
	//  3. Set the tenant: tx.Exec with "SELECT set_config('app.pharmacy_id',
	//     $1, true)" and pharmacyID.String() as $1. On an error, return it
	//     wrapped, "setting the tenant: %w".
	//  4. Call fn(tx). If it returns an error, return that error as it is,
	//     not wrapped, so callers can inspect it (a *pgconn.PgError, say).
	//  5. Commit: tx.Commit(ctx). Return its error wrapped, "committing: %w",
	//     or nil.
	//
	// The tests in tenant_test.go say what's expected; from backend/,
	// `go test ./internal/platform/db/` runs them. (Replace "errors" with
	// "fmt" in the imports; gopls does it on save.)
	return errors.New("WithTenant isn't written yet")
}
