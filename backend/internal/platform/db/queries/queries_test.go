package queries_test

import (
	"context"
	"errors"
	"log/slog"
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

// pharmacy creates a pharmacy called name.
func pharmacy(t *testing.T, q *queries.Queries, name string) queries.Pharmacy {
	t.Helper()
	p, err := q.CreatePharmacy(context.Background(), name)
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
	p := pharmacy(t, q, "Farmàcia del Sol")

	if p.ID == uuid.Nil {
		t.Error("ID is empty; want the one Postgres generated")
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

// YOUR TURN (milestone 2.3). This test is empty: write it. It checks two
// things about ListUsers (in users.sql): it returns only the users of the
// pharmacy it's given, and it returns them in order of name.
//
//  1. Make the queries: q := newQueries(t).
//  2. Make two pharmacies with the pharmacy helper above, say sol and lluna.
//  3. Make three users with the user helper: "Marta" and then "Jordi" in
//     sol (in that order, so the list has to sort them), and "Pau" in lluna.
//  4. Call q.ListUsers(context.Background(), sol.ID). It returns a slice of
//     users and an error: if the error isn't nil, t.Fatalf with it.
//  5. Collect the users' names in a []string with a for loop (for _, u :=
//     range users), and compare it with []string{"Jordi", "Marta"} using
//     slices.Equal (add "slices" to the imports). If they're different,
//     t.Errorf with both, so a failure says what came back.
//
// Then delete the t.Fatal line.
func TestListUsersReturnsOnePharmacysUsersByName(t *testing.T) {
	t.Fatal("your turn: write this test (see the comment above it)")
}
