// Package migrations holds this service's versioned SQL schema migrations
// (embedded into the binary) plus the runner that applies them at startup.
// Replaces GORM's AutoMigrate — 000001_init mirrors exactly the schema
// AutoMigrate had already created, so adopting this tool is a pure tooling
// swap with zero schema change.
package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var files embed.FS

// Run applies any pending migrations against an already-open connection.
// Safe to call on every startup — migrate tracks the applied version in
// its own schema_migrations table and no-ops if already current.
//
// adoptionCheckTable names one table created by migration 000001. If a
// database predates this tool (schema already created by GORM's
// AutoMigrate, no schema_migrations row yet) but that table already
// exists, 000001 is marked applied via Force instead of re-run — running
// its CREATE TABLE against tables that already exist would otherwise
// fail every such database once on adoption.
func Run(sqlDB *sql.DB, adoptionCheckTable string) error {
	source, err := iofs.New(files, ".")
	if err != nil {
		return err
	}

	dbDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", dbDriver)
	if err != nil {
		return err
	}

	version, dirty, err := m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		exists, checkErr := tableExists(sqlDB, adoptionCheckTable)
		if checkErr != nil {
			return checkErr
		}
		if exists {
			if err := m.Force(1); err != nil {
				return err
			}
		}
	case err != nil:
		return err
	case dirty:
		return fmt.Errorf("database is in a dirty migration state at version %d", version)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func tableExists(db *sql.DB, name string) (bool, error) {
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`,
		name,
	).Scan(&exists)
	return exists, err
}
