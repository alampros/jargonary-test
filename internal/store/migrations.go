package store

import (
	"context"
	"embed"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// The embedded initial schema is also sqlc's input. Goose applies it and its
// version record transactionally. No separate CLI is needed at startup.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

func (s *Store) migrate(ctx context.Context) error {
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, files, goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return err
	}
	if _, err = provider.Up(ctx); err != nil {
		return err
	}
	return s.initializeSourceLetters(ctx)
}
