package db_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db"
	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
)

func TestOpenConnectsToTheDatabase(t *testing.T) {
	url := dbtest.NewDatabase(t)
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()

	var name string
	if err := pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&name); err != nil {
		t.Fatalf("query: %v", err)
	}
	if !strings.Contains(url, "/"+name+"?") {
		t.Errorf("connected to %q, want the database in %s", name, url)
	}
}

func TestOpenFailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	// Port 1: nothing listens there, so the connection is refused at once.
	_, err := db.Open(context.Background(), "postgres://turnia:turnia@127.0.0.1:1/turnia?sslmode=disable&connect_timeout=2")
	if err == nil {
		t.Fatal("Open succeeded with no database to connect to")
	}
	if !strings.HasPrefix(err.Error(), "connecting: ") {
		t.Errorf("err = %v, want it to say it was connecting", err)
	}
}

func TestOpenRejectsAnInvalidURL(t *testing.T) {
	_, err := db.Open(context.Background(), "postgres://turnia@localhost:5433/turnia?pool_max_conns=many")
	if err == nil || !strings.HasPrefix(err.Error(), "configuring the pool: ") {
		t.Errorf("err = %v, want a configuration error", err)
	}
}
