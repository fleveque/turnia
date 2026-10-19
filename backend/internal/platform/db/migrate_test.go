package db_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
	"github.com/jackc/pgx/v5"
)

func TestMigrateCreatesTheTables(t *testing.T) {
	url := dbtest.NewDatabase(t)
	ctx := context.Background()
	if err := db.Migrate(ctx, url, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var n int
	err = conn.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name IN ('pharmacies', 'users')`).Scan(&n)
	if err != nil || n != 2 {
		t.Errorf("found %d of the 2 tables (err %v)", n, err)
	}
}

func TestMigrateTwiceIsUpToDate(t *testing.T) {
	url := dbtest.NewDatabase(t)
	ctx := context.Background()
	if err := db.Migrate(ctx, url, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	var logs bytes.Buffer
	if err := db.Migrate(ctx, url, slog.New(slog.NewTextHandler(&logs, nil))); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if !strings.Contains(logs.String(), "database up to date") || strings.Contains(logs.String(), "msg=migrated") {
		t.Errorf("second run should change nothing; logged:\n%s", logs.String())
	}
}
