// Package db is Turnia's access to Postgres. For now: the migrations, which
// are SQL files built into the binary, and the code that applies them.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	// Registers the "pgx" driver with database/sql, which goose uses.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// The migrations directory, compiled into the binary: a deployed turnia
// needs no files next to it.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies every migration that hasn't been applied yet, in order,
// and logs each one. A database that's already up to date is not an error.
//
// Two processes migrating at once (two containers starting during a deploy)
// take turns: goose holds a Postgres advisory lock while it works.
func Migrate(ctx context.Context, databaseURL string, log *slog.Logger) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("opening the database: %w", err)
	}
	defer sqlDB.Close()

	// The files live under migrations/ inside the embedded tree; goose wants
	// them at the top.
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, files, goose.WithSessionLocker(locker))
	if err != nil {
		return err
	}

	results, err := provider.Up(ctx)
	for _, r := range results {
		log.Info("migrated", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	if err != nil {
		return fmt.Errorf("migrating: %w", err)
	}
	if len(results) == 0 {
		log.Info("database up to date")
	}
	return nil
}
