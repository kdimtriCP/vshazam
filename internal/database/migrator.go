package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

type Migration struct {
	Version string
	Name    string
	SQL     string
}

type Migrator struct {
	db     *sql.DB
	dbType string
}

func NewMigrator(db *sql.DB, dbType string) *Migrator {
	return &Migrator{
		db:     db,
		dbType: dbType,
	}
}

// Initialize creates the migrations tracking table if it doesn't exist
func (m *Migrator) Initialize() error {
	if m.dbType != "postgres" {
		// Skip migrations for SQLite as tables are created directly
		return nil
	}

	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	log.Println("Migration tracking table ready")
	return nil
}

// GetAppliedMigrations returns a list of already applied migration versions
func (m *Migrator) GetAppliedMigrations() (map[string]bool, error) {
	applied := make(map[string]bool)

	rows, err := m.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	return applied, nil
}

// LoadMigrations loads all migration files from the migrations directory
func (m *Migrator) LoadMigrations(migrationsPath string) ([]Migration, error) {
	files, err := ioutil.ReadDir(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Parse version from filename (e.g., "001_init.sql" -> "001")
		parts := strings.Split(file.Name(), "_")
		if len(parts) < 2 {
			log.Printf("Skipping invalid migration filename: %s", file.Name())
			continue
		}
		version := parts[0]

		// Read migration content
		content, err := ioutil.ReadFile(filepath.Join(migrationsPath, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    file.Name(),
			SQL:     string(content),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// ApplyMigration runs a single migration
func (m *Migrator) ApplyMigration(migration Migration) error {
	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
	}

	// Record migration as applied
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		migration.Version,
	); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", migration.Name, err)
	}

	log.Printf("Applied migration: %s", migration.Name)
	return nil
}

// Run executes all pending migrations
func (m *Migrator) Run(migrationsPath string) error {
	if m.dbType != "postgres" {
		log.Println("Skipping migrations for non-PostgreSQL database")
		return nil
	}

	// Initialize migrations table
	if err := m.Initialize(); err != nil {
		return err
	}

	// Get already applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	// Load all migrations
	migrations, err := m.LoadMigrations(migrationsPath)
	if err != nil {
		return err
	}

	// Apply pending migrations
	pendingCount := 0
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}

		if err := m.ApplyMigration(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		pendingCount++
	}

	if pendingCount == 0 {
		log.Println("No pending migrations")
	} else {
		log.Printf("Successfully applied %d migration(s)", pendingCount)
	}

	return nil
}