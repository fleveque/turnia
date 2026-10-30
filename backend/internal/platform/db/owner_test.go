package db_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ownedTenants is tenants as production has it: a database owned by
// turnia_owner, migrated by it, and with the same pharmacies and users. It
// returns a pool connected as turnia_owner. The rows go in one pharmacy at a
// time, because the policies apply to the owner too.
func ownedTenants(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := dbtest.NewOwnedDatabase(t)
	ctx := context.Background()
	if err := db.Migrate(ctx, url, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Migrate as turnia_owner: %v", err)
	}
	owner, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { closePool(t, owner) })

	add := func(pharmacy, sql string) {
		t.Helper()
		err := db.WithTenant(ctx, owner, uuid.MustParse(pharmacy), func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql)
			return err
		})
		if err != nil {
			t.Fatalf("adding %s: %v", pharmacy, err)
		}
	}
	add(sol, `
		INSERT INTO pharmacies (id, name) VALUES ('`+sol+`', 'Farmàcia del Sol');
		INSERT INTO users (pharmacy_id, email, name, role) VALUES
			('`+sol+`', 'laia@example.com', 'Laia', 'admin'),
			('`+sol+`', 'pau@example.com', 'Pau', 'employee')`)
	add(mar, `
		INSERT INTO pharmacies (id, name) VALUES ('`+mar+`', 'Farmàcia del Mar');
		INSERT INTO users (pharmacy_id, email, name, role) VALUES
			('`+mar+`', 'marta@example.com', 'Marta', 'admin')`)
	return owner
}

// turnia_owner has no attribute that would skip row-level security or reach
// outside its own database. deploy/postgres/01-roles.sql says so, and this
// checks the role the server really has, even one made by an older script.
func TestTheOwnerIsAnOrdinaryRole(t *testing.T) {
	owner := ownedTenants(t)
	var special bool
	err := owner.QueryRow(context.Background(), `
		SELECT rolsuper OR rolbypassrls OR rolcreatedb OR rolcreaterole
		FROM pg_roles WHERE rolname = current_user`).Scan(&special)
	if err != nil {
		t.Fatalf("reading the role: %v", err)
	}
	if special {
		t.Error("turnia_owner has a special attribute, want none")
	}
}

// FORCE ROW LEVEL SECURITY puts the tables' owner under the policies too.
// This is the test 3.2 couldn't have: there, the owner was a superuser, and
// a superuser skips the policies whether FORCE is there or not.
func TestForceFiltersTheOwner(t *testing.T) {
	owner := ownedTenants(t)

	// YOUR TURN (milestone 4.2): the assertions, in three steps. No new Go:
	// TestATenantSeesOnlyItsOwnRows in tenancy_test.go has the same shape.
	//
	//  1. With no tenant, a query is an error. owner.Exec(ctx, sql) returns
	//     two values, and the second is the error. Use
	//     context.Background() for ctx and "SELECT * FROM users" for sql,
	//     and if the error is nil, report it with t.Error.
	//  2. inTenant(t, owner, sol, "SELECT * FROM users") returns the rows and
	//     an error. It should be 2 rows and no error; report anything else
	//     with t.Errorf. The same as mar should be 1 row.
	//  3. As sol, this changes nothing, because sol can't see mar's users:
	//     "UPDATE users SET locale = 'ca' WHERE pharmacy_id = '" + mar + "'"
	//     should be 0 rows and no error.
	//
	// Then check the test can fail: delete the line
	// "ALTER TABLE users FORCE ROW LEVEL SECURITY;" from
	// migrations/00003_tenant_policies.sql, and from backend/ run
	// `go test ./internal/platform/db/ -run TestForceFiltersTheOwner`. Each
	// step should report a failure. Put the line back afterwards.
	_ = owner
	t.Fatal("not written yet: see YOUR TURN")
}
