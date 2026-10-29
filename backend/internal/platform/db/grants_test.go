package db_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
	"github.com/jackc/pgx/v5/pgconn"
)

// The server connects as turnia_app, which can use the tables' rows but not
// change the tables: a bug, or an injected statement, can't drop one.
func TestTheAppRoleUsesRowsButCantChangeTheSchema(t *testing.T) {
	url := dbtest.NewDatabase(t)
	ctx := context.Background()
	if err := db.Migrate(ctx, url, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	app, err := db.Open(ctx, dbtest.AppURL(t, url))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer app.Close()

	allowed := []string{
		"INSERT INTO pharmacies (name) VALUES ('Farmàcia del Sol')",
		"SELECT * FROM pharmacies",
		"UPDATE pharmacies SET swaps_enabled = true",
		"SELECT * FROM users",
		"DELETE FROM pharmacies",
	}
	for _, sql := range allowed {
		if _, err := app.Exec(ctx, sql); err != nil {
			t.Errorf("%s: %v", sql, err)
		}
	}

	refused := []string{
		"CREATE TABLE notes (id int)",
		"ALTER TABLE users ADD COLUMN notes text",
		"TRUNCATE users",
		"DROP TABLE users",
	}
	for _, sql := range refused {
		_, err := app.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		// 42501 is insufficient_privilege.
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: err = %v, want insufficient_privilege (42501)", sql, err)
		}
	}
}
