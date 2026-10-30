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

// defaultServer is the development Postgres from compose.yaml (`make db-up`).
const defaultServer = "postgres://turnia:turnia@localhost:5433/turnia_test?sslmode=disable"

// NewDatabase creates an empty database for this test, drops it when the
// test ends, and returns its connection URL.
//
// It connects to the server named by TURNIA_TEST_DATABASE_URL (CI sets it),
// or to the development Postgres. If the server isn't there, the test fails:
// a skipped test reports ok, which would hide a stopped database.
func NewDatabase(t *testing.T) string {
	t.Helper()
	return create(t, "")
}

// NewOwnedDatabase is NewDatabase with the database production has: owned by
// turnia_owner, a role with no special attributes, and the returned URL
// connects as that role. Row-level security applies to it, unlike the
// superuser that owns NewDatabase's databases.
func NewOwnedDatabase(t *testing.T) string {
	t.Helper()
	return withUser(t, create(t, "turnia_owner"), "turnia_owner", "turnia_owner")
}

// create makes the database, owned by owner, or by the user connecting if
// owner is empty.
func create(t *testing.T, owner string) string {
	t.Helper()
	server := os.Getenv("TURNIA_TEST_DATABASE_URL")
	if server == "" {
		server = defaultServer
	}

	name := fmt.Sprintf("turnia_test_%x", rand.Uint64())
	sql := "CREATE DATABASE " + pgx.Identifier{name}.Sanitize()
	if owner != "" {
		sql += " OWNER " + pgx.Identifier{owner}.Sanitize()
	}
	exec(t, server, sql)
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

// AppURL is databaseURL with the user the server connects as, turnia_app,
// and its development password (docker-compose.yml). Tests that should see
// what the server sees connect with it.
func AppURL(t *testing.T, databaseURL string) string {
	t.Helper()
	return withUser(t, databaseURL, "turnia_app", "turnia_app")
}

func withUser(t *testing.T, databaseURL, user, password string) string {
	t.Helper()
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parsing %q: %v", databaseURL, err)
	}
	u.User = url.UserPassword(user, password)
	return u.String()
}

func exec(t *testing.T, server, sql string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, server)
	if err != nil {
		t.Fatalf("connecting to the test server (is it running? make db-up): %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
