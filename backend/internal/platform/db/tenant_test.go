package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/queries"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// countUsers returns how many users the transaction can see.
func countUsers(t *testing.T, tx pgx.Tx) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatalf("counting users: %v", err)
	}
	return n
}

// addUser adds an employee to the transaction's pharmacy.
func addUser(tx pgx.Tx, pharmacyID uuid.UUID, name string) error {
	_, err := queries.New(tx).CreateUser(context.Background(), queries.CreateUserParams{
		PharmacyID: pharmacyID, Email: name + "@example.com", Name: name, Role: "employee",
	})
	return err
}

func TestWithTenantSeesOnlyThatPharmacy(t *testing.T) {
	app := tenants(t)
	for _, tt := range []struct {
		pharmacy string
		users    int
	}{{sol, 2}, {mar, 1}} {
		err := db.WithTenant(context.Background(), app, uuid.MustParse(tt.pharmacy), func(tx pgx.Tx) error {
			if n := countUsers(t, tx); n != tt.users {
				t.Errorf("as %s, %d users, want %d", tt.pharmacy, n, tt.users)
			}
			return nil
		})
		if err != nil {
			t.Errorf("WithTenant(%s): %v", tt.pharmacy, err)
		}
	}
}

func TestWithTenantCommitsWhenFnSucceeds(t *testing.T) {
	app := tenants(t)
	ctx := context.Background()
	id := uuid.MustParse(sol)
	if err := db.WithTenant(ctx, app, id, func(tx pgx.Tx) error { return addUser(tx, id, "joan") }); err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	db.WithTenant(ctx, app, id, func(tx pgx.Tx) error {
		if n := countUsers(t, tx); n != 3 {
			t.Errorf("%d users after adding one to 2, want 3", n)
		}
		return nil
	})
}

func TestWithTenantRollsBackAndReturnsFnsError(t *testing.T) {
	app := tenants(t)
	ctx := context.Background()
	id := uuid.MustParse(sol)
	boom := errors.New("boom")
	err := db.WithTenant(ctx, app, id, func(tx pgx.Tx) error {
		if err := addUser(tx, id, "joan"); err != nil {
			t.Fatalf("addUser: %v", err)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want fn's error", err)
	}
	db.WithTenant(ctx, app, id, func(tx pgx.Tx) error {
		if n := countUsers(t, tx); n != 2 {
			t.Errorf("%d users, want 2: the user added before the error should be gone", n)
		}
		return nil
	})
}

// A pooled connection serves one request after another. The tenant of one
// must not still be set for the next.
func TestWithTenantLeavesNoTenantOnTheConnection(t *testing.T) {
	app := tenants(t)
	ctx := context.Background()
	// A pool of one connection, so every use below is the same connection.
	cfg := app.Config()
	cfg.MaxConns = 1
	one, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	t.Cleanup(func() { closePool(t, one) })

	id := uuid.MustParse(sol)
	db.WithTenant(ctx, one, id, func(tx pgx.Tx) error { return nil })
	db.WithTenant(ctx, one, id, func(tx pgx.Tx) error { return errors.New("boom") })
	// Otherwise the queries below would wait for the connection forever.
	if n := one.Stat().AcquiredConns(); n != 0 {
		t.Fatalf("%d connections still in use after WithTenant returned, want 0", n)
	}

	var tenant string
	if err := one.QueryRow(ctx, "SELECT current_setting('app.pharmacy_id')").Scan(&tenant); err != nil {
		t.Fatalf("reading the setting: %v", err)
	}
	if tenant != "" {
		t.Errorf("app.pharmacy_id = %q after WithTenant, want it empty", tenant)
	}
	if _, err := one.Exec(ctx, "SELECT * FROM users"); err == nil {
		t.Error("a query after WithTenant, without it: err = nil, want an error")
	}
}

// Creating a pharmacy as turnia_app works only in a transaction whose tenant
// is the new pharmacy's id, which is why the id comes from Go.
func TestWithTenantCreatesAPharmacy(t *testing.T) {
	app := tenants(t)
	id := uuid.Must(uuid.NewV7())
	err := db.WithTenant(context.Background(), app, id, func(tx pgx.Tx) error {
		_, err := queries.New(tx).CreatePharmacy(context.Background(), queries.CreatePharmacyParams{ID: id, Name: "Farmàcia Nova"})
		return err
	})
	if err != nil {
		t.Errorf("WithTenant(new id): %v", err)
	}
}
