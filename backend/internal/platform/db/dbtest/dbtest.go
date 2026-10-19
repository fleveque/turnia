// Package dbtest gives each test its own empty Postgres database, so tests
// can run in parallel and never see each other's rows.
package dbtest

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// NewDatabase creates an empty database for this test, drops it when the
// test ends, and returns its connection URL.
//
// It connects to the server named by TURNIA_TEST_DATABASE_URL (make test and
// CI set it); without it, the test is skipped.
func NewDatabase(t *testing.T) string {
	t.Helper()
	server := os.Getenv("TURNIA_TEST_DATABASE_URL")
	if server == "" {
		t.Skip("TURNIA_TEST_DATABASE_URL isn't set; `make test` sets it")
	}

	name := fmt.Sprintf("turnia_test_%x", rand.Uint64())
	exec(t, server, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	t.Cleanup(func() {
		// FORCE closes connections the test left open.
		exec(t, server, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
	})

	u, err := url.Parse(server)
	if err != nil {
		t.Fatalf("TURNIA_TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

func exec(t *testing.T, server, sql string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, server)
	if err != nil {
		t.Fatalf("connecting to the test server: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
