package db_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Two pharmacies, as literals so the SQL in the tests can name them.
const (
	sol = "00000000-0000-7000-8000-000000000001"
	mar = "00000000-0000-7000-8000-000000000002"
)

// tenants migrates a database of this test's own, adds the pharmacies sol,
// with two users, and mar, with one, and returns a pool connected as
// turnia_app. The rows are added as the owner, a superuser in development
// and CI, which row-level security doesn't apply to.
func tenants(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := dbtest.NewDatabase(t)
	ctx := context.Background()
	if err := db.Migrate(ctx, url, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	owner, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer owner.Close()
	_, err = owner.Exec(ctx, `
		INSERT INTO pharmacies (id, name) VALUES
			('`+sol+`', 'Farmàcia del Sol'), ('`+mar+`', 'Farmàcia del Mar');
		INSERT INTO users (pharmacy_id, email, name, role) VALUES
			('`+sol+`', 'laia@example.com', 'Laia', 'admin'),
			('`+sol+`', 'pau@example.com', 'Pau', 'employee'),
			('`+mar+`', 'marta@example.com', 'Marta', 'admin')`)
	if err != nil {
		t.Fatalf("adding the pharmacies: %v", err)
	}

	app, err := db.Open(ctx, dbtest.AppURL(t, url))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(app.Close)
	return app
}

// inTenant runs sql with app.pharmacy_id set to pharmacyID, in a transaction
// it rolls back, and returns how many rows the statement returned or changed.
func inTenant(t *testing.T, app *pgxpool.Pool, pharmacyID, sql string) (int64, error) {
	t.Helper()
	ctx := context.Background()
	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx)
	// true: the setting ends with the transaction.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.pharmacy_id', $1, true)", pharmacyID); err != nil {
		t.Fatalf("set_config: %v", err)
	}
	tag, err := tx.Exec(ctx, sql)
	return tag.RowsAffected(), err
}

// With no tenant, a query is an error, not an empty result: a missing
// WithTenant shows up the first time the code runs.
func TestAQueryWithoutATenantFails(t *testing.T) {
	app := tenants(t)
	queries := []string{"SELECT * FROM users", "SELECT * FROM pharmacies"}
	// First, on connections where app.pharmacy_id was never set.
	for _, sql := range queries {
		if _, err := app.Exec(context.Background(), sql); err == nil {
			t.Errorf("%s with no tenant: err = nil, want an error", sql)
		}
	}
	// Then '', what current_setting returns once a transaction's setting ended.
	for _, sql := range queries {
		if _, err := inTenant(t, app, "", sql); err == nil {
			t.Errorf("%s with an empty tenant: err = nil, want an error", sql)
		}
	}
}

func TestATenantSeesOnlyItsOwnRows(t *testing.T) {
	app := tenants(t)
	tests := []struct {
		tenant, sql string
		rows        int64
	}{
		{sol, "SELECT * FROM users", 2},
		{mar, "SELECT * FROM users", 1},
		{sol, "SELECT * FROM users WHERE pharmacy_id = '" + mar + "'", 0},
		{sol, "SELECT * FROM pharmacies", 1},
		{sol, "SELECT * FROM pharmacies WHERE id = '" + sol + "'", 1},
		{mar, "SELECT * FROM pharmacies WHERE id = '" + mar + "'", 1},
	}
	for _, tt := range tests {
		rows, err := inTenant(t, app, tt.tenant, tt.sql)
		if err != nil || rows != tt.rows {
			t.Errorf("as %s, %s: %d rows, err = %v; want %d rows", tt.tenant, tt.sql, rows, err, tt.rows)
		}
	}
}

func TestATenantCantWriteToAnother(t *testing.T) {
	app := tenants(t)
	// Writing to its own rows works...
	allowed := []struct {
		sql  string
		rows int64
	}{
		{"INSERT INTO users (pharmacy_id, email, name, role) VALUES ('" + sol + "', 'joan@example.com', 'Joan', 'employee')", 1},
		{"UPDATE users SET locale = 'ca'", 2},
		{"UPDATE pharmacies SET swaps_enabled = true", 1},
		// ...and another's rows are invisible, so nothing matches.
		{"UPDATE users SET locale = 'ca' WHERE pharmacy_id = '" + mar + "'", 0},
		{"DELETE FROM users WHERE pharmacy_id = '" + mar + "'", 0},
		{"UPDATE pharmacies SET name = 'Farmàcia' WHERE id = '" + mar + "'", 0},
	}
	for _, tt := range allowed {
		rows, err := inTenant(t, app, sol, tt.sql)
		if err != nil || rows != tt.rows {
			t.Errorf("as sol, %s: %d rows, err = %v; want %d rows", tt.sql, rows, err, tt.rows)
		}
	}

	// A row written into another pharmacy is refused outright.
	refused := []string{
		"INSERT INTO users (pharmacy_id, email, name, role) VALUES ('" + mar + "', 'joan@example.com', 'Joan', 'employee')",
		"UPDATE users SET pharmacy_id = '" + mar + "'",
		"INSERT INTO pharmacies (id, name) VALUES ('00000000-0000-7000-8000-000000000003', 'Farmàcia Nova')",
	}
	for _, sql := range refused {
		if _, err := inTenant(t, app, sol, sql); err == nil {
			t.Errorf("as sol, %s: err = nil, want it refused", sql)
		}
	}
}
