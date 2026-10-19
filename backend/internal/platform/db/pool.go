package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open returns a connection pool for databaseURL, after checking that the
// database answers. A server that can't reach its database fails at start,
// where the deploy sees it, rather than on its first request.
//
// The pool opens connections as requests need them, up to its maximum
// (pgxpool's default: 4, or the number of CPUs if that's more; the URL can
// set pool_max_conns). Close it when the server stops.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("configuring the pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting: %w", err)
	}
	return pool, nil
}
