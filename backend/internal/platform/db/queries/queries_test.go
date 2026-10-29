package queries_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
	"github.com/fleveque/turnia/backend/internal/platform/db/queries"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// newQueries returns the queries on a fresh, migrated database of this
// test's own.
func newQueries(t *testing.T) *queries.Queries {
	t.Helper()
	url := dbtest.NewDatabase(t)
	ctx := context.Background()
	if err := db.Migrate(ctx, url, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(pool.Close)
	return queries.New(pool)
}

// pharmacy creates a pharmacy called name, with a new id.
func pharmacy(t *testing.T, q *queries.Queries, name string) queries.Pharmacy {
	t.Helper()
	p, err := q.CreatePharmacy(context.Background(), queries.CreatePharmacyParams{
		ID:   uuid.Must(uuid.NewV7()),
		Name: name,
	})
	if err != nil {
		t.Fatalf("CreatePharmacy(%q): %v", name, err)
	}
	return p
}

// user creates an employee called name in the pharmacy, with an email made
// from the name.
func user(t *testing.T, q *queries.Queries, pharmacyID uuid.UUID, name string) queries.User {
	t.Helper()
	u, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		PharmacyID: pharmacyID,
		Email:      strings.ToLower(name) + "@example.com",
		Name:       name,
		Role:       "employee",
	})
	if err != nil {
		t.Fatalf("CreateUser(%q): %v", name, err)
	}
	return u
}

func TestCreatePharmacyFillsInTheDefaults(t *testing.T) {
	q := newQueries(t)
	before := time.Now()
	id := uuid.Must(uuid.NewV7())
	p, err := q.CreatePharmacy(context.Background(), queries.CreatePharmacyParams{ID: id, Name: "Farmàcia del Sol"})
	if err != nil {
		t.Fatalf("CreatePharmacy: %v", err)
	}

	if p.ID != id {
		t.Errorf("ID = %v, want the one given, %v", p.ID, id)
	}
	if p.Name != "Farmàcia del Sol" || p.Timezone != "Europe/Madrid" || p.DefaultLocale != "es" || p.SwapsEnabled {
		t.Errorf("got %+v, want the name given and the table's defaults", p)
	}
	// Postgres's clock and the test's are the same machine's, or close.
	if p.CreatedAt.Before(before.Add(-time.Minute)) || p.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("CreatedAt = %v, want about now (%v)", p.CreatedAt, before)
	}
}

func TestCreateUserRejectsAnEmailThatDiffersOnlyInCase(t *testing.T) {
	q := newQueries(t)
	p := pharmacy(t, q, "Farmàcia del Sol")
	user(t, q, p.ID, "Laia") // laia@example.com

	_, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		PharmacyID: p.ID, Email: "Laia@Example.com", Name: "Laia", Role: "employee",
	})
	// 23505 is Postgres's code for unique_violation.
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Errorf("err = %v, want a unique violation (23505)", err)
	}
}

func TestListUsersReturnsOnePharmacysUsersByName(t *testing.T) {
	q := newQueries(t)
	sol := pharmacy(t, q, "Farmàcia del Sol")
	user(t, q, sol.ID, "Marta")
	user(t, q, sol.ID, "Jordi")
	lluna := pharmacy(t, q, "Farmàcia de la Lluna")
	user(t, q, lluna.ID, "Pau")

	users, err := q.ListUsers(context.Background(), sol.ID)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}

	names := []string{}
	for _, u := range users {
		names = append(names, u.Name)
	}

	expected := []string{"Jordi", "Marta"}
	if !slices.Equal(names, expected) {
		t.Errorf("names = %v, want %v", names, expected)
	}
}
