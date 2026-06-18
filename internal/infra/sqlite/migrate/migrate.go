// Package migrate runs versioned SQLite migrations embedded in the binary.
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Up applies all pending migrations to the given database.
func Up(ctx context.Context, db *sql.DB) error {
	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations filesystem: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
