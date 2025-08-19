package database

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator handles database migrations
type Migrator struct {
	migrate *migrate.Migrate
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *DB, migrationsPath string) (*Migrator, error) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{migrate: m}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if err := m.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Database migrations completed successfully")
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	if err := m.migrate.Down(); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	log.Println("Database migrations rolled back successfully")
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}
	if dirty {
		return version, fmt.Errorf("database is in dirty state at version %d", version)
	}
	return version, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	log.Printf("Forced migration version to %d", version)
	return nil
}

// Drop drops the entire database schema
func (m *Migrator) Drop() error {
	if err := m.migrate.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	log.Println("Database schema dropped successfully")
	return nil
}

// Close closes the migrator
func (m *Migrator) Close() error {
	_, err := m.migrate.Close()
	if err != nil {
		return fmt.Errorf("failed to close migrator: %w", err)
	}
	return nil
}
