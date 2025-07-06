package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version     string
	Description string
	Module      string
	UpSQL       string
	DownSQL     string
	FilePath    string
}

// MigrationManager handles database migrations
type MigrationManager struct {
	db            *sql.DB
	migrationsDir string
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{
		db:            db,
		migrationsDir: "database/migrations",
	}
}

// RunMigrations executes all pending migrations
func (m *MigrationManager) RunMigrations() error {
	// Create migrations table if not exists
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Load migrations from files
	migrations, err := m.loadMigrationsFromFiles()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		vi, _ := strconv.Atoi(migrations[i].Version)
		vj, _ := strconv.Atoi(migrations[j].Version)
		return vi < vj
	})

	// Execute pending migrations
	for _, migration := range migrations {
		if err := m.executeMigration(migration); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}
	}

	log.Println("All migrations executed successfully")
	return nil
}

// createMigrationsTable creates the migrations tracking table
func (m *MigrationManager) createMigrationsTable() error {
	// Create table with new structure
	query := `
	CREATE TABLE IF NOT EXISTS public.migrations (
		version VARCHAR(255) PRIMARY KEY,
		description TEXT,
		module VARCHAR(100),
		file_path VARCHAR(500),
		executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := m.db.Exec(query); err != nil {
		return err
	}

	// Check if we need to migrate old migration table structure
	if err := m.migrateMigrationsTable(); err != nil {
		return fmt.Errorf("failed to migrate migrations table: %w", err)
	}

	return nil
}

// migrateMigrationsTable handles migration table structure evolution
func (m *MigrationManager) migrateMigrationsTable() error {
	// Check if module column exists
	var exists bool
	err := m.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'migrations' 
			AND column_name = 'module'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check column existence: %w", err)
	}

	// If module column doesn't exist, add it
	if !exists {
		log.Println("Migrating migrations table structure...")

		// Add module column
		if _, err := m.db.Exec("ALTER TABLE public.migrations ADD COLUMN module VARCHAR(100)"); err != nil {
			return fmt.Errorf("failed to add module column: %w", err)
		}

		// Add file_path column
		if _, err := m.db.Exec("ALTER TABLE public.migrations ADD COLUMN file_path VARCHAR(500)"); err != nil {
			return fmt.Errorf("failed to add file_path column: %w", err)
		}

		// Update existing records with default module
		if _, err := m.db.Exec("UPDATE public.migrations SET module = 'legacy' WHERE module IS NULL"); err != nil {
			return fmt.Errorf("failed to update existing records: %w", err)
		}

		log.Println("Migration table structure updated successfully")
	}

	return nil
}

// loadMigrationsFromFiles reads migration files from filesystem
func (m *MigrationManager) loadMigrationsFromFiles() ([]Migration, error) {
	var migrations []Migration

	// Check if migrations directory exists
	if _, err := os.Stat(m.migrationsDir); os.IsNotExist(err) {
		log.Printf("Migrations directory %s does not exist, creating it...", m.migrationsDir)
		if err := m.createMigrationDirectories(); err != nil {
			return nil, fmt.Errorf("failed to create migration directories: %w", err)
		}
		if err := m.createDefaultMigrationFiles(); err != nil {
			return nil, fmt.Errorf("failed to create default migration files: %w", err)
		}
	}

	// Walk through migration directories
	err := filepath.WalkDir(m.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(path, ".sql") {
			return nil
		}

		migration, err := m.parseMigrationFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse migration file %s: %w", path, err)
		}

		migrations = append(migrations, migration)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk migration directory: %w", err)
	}

	return migrations, nil
}

// parseMigrationFile parses a single migration file
func (m *MigrationManager) parseMigrationFile(filePath string) (Migration, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read file: %w", err)
	}

	// Extract version and description from filename
	filename := filepath.Base(filePath)
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	version := parts[0]
	description := strings.TrimSuffix(parts[1], ".sql")
	description = strings.ReplaceAll(description, "_", " ")

	// Extract module from directory path
	dir := filepath.Dir(filePath)
	module := filepath.Base(dir)

	// Split content into UP and DOWN sections
	upSQL, downSQL := m.splitMigrationContent(string(content))

	return Migration{
		Version:     version,
		Description: description,
		Module:      module,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
		FilePath:    filePath,
	}, nil
}

// splitMigrationContent splits migration content into UP and DOWN sections
func (m *MigrationManager) splitMigrationContent(content string) (string, string) {
	lines := strings.Split(content, "\n")
	var upLines, downLines []string
	var inDownSection bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "-- DOWN") || strings.HasPrefix(trimmed, "--DOWN") {
			inDownSection = true
			continue
		}

		if strings.HasPrefix(trimmed, "-- UP") || strings.HasPrefix(trimmed, "--UP") {
			inDownSection = false
			continue
		}

		if inDownSection {
			downLines = append(downLines, line)
		} else {
			upLines = append(upLines, line)
		}
	}

	return strings.Join(upLines, "\n"), strings.Join(downLines, "\n")
}

// executeMigration executes a single migration if not already applied
func (m *MigrationManager) executeMigration(migration Migration) error {
	// Check if migration already executed
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM public.migrations WHERE version = $1", migration.Version).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	// Skip if already executed
	if count > 0 {
		log.Printf("Migration %s (%s) already executed, skipping", migration.Version, migration.Module)
		return nil
	}

	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration
	if strings.TrimSpace(migration.UpSQL) != "" {
		if _, err := tx.Exec(migration.UpSQL); err != nil {
			return fmt.Errorf("failed to execute migration SQL: %w", err)
		}
	}

	// Record migration
	if _, err := tx.Exec(
		"INSERT INTO public.migrations (version, description, module, file_path) VALUES ($1, $2, $3, $4)",
		migration.Version, migration.Description, migration.Module, migration.FilePath,
	); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	log.Printf("Migration %s executed successfully: %s [%s]", migration.Version, migration.Description, migration.Module)
	return nil
}

// Rollback rolls back the last migration
func (m *MigrationManager) Rollback() error {
	// Get last migration
	var version, description, module, filePath string
	err := m.db.QueryRow(`
		SELECT version, description, module, file_path 
		FROM public.migrations 
		ORDER BY executed_at DESC 
		LIMIT 1
	`).Scan(&version, &description, &module, &filePath)

	if err == sql.ErrNoRows {
		log.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	// Load migration file to get DOWN SQL
	migration, err := m.parseMigrationFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse migration file for rollback: %w", err)
	}

	// Execute rollback
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute down migration
	if strings.TrimSpace(migration.DownSQL) != "" {
		if _, err := tx.Exec(migration.DownSQL); err != nil {
			return fmt.Errorf("failed to execute rollback SQL: %w", err)
		}
	}

	// Remove migration record
	if _, err := tx.Exec("DELETE FROM public.migrations WHERE version = $1", version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	log.Printf("Migration %s rolled back successfully: %s [%s]", version, description, module)
	return nil
}

// GetMigrationStatus returns current migration status
func (m *MigrationManager) GetMigrationStatus() ([]map[string]interface{}, error) {
	rows, err := m.db.Query(`
		SELECT version, description, module, executed_at 
		FROM public.migrations 
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var status []map[string]interface{}
	for rows.Next() {
		var version, description, module, executedAt string
		if err := rows.Scan(&version, &description, &module, &executedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		status = append(status, map[string]interface{}{
			"version":     version,
			"description": description,
			"module":      module,
			"executed_at": executedAt,
		})
	}

	return status, nil
}

// createMigrationDirectories creates the migration directory structure
func (m *MigrationManager) createMigrationDirectories() error {
	dirs := []string{
		filepath.Join(m.migrationsDir, "user_management"),
		filepath.Join(m.migrationsDir, "sensor_data"),
		filepath.Join(m.migrationsDir, "cross_module"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// createDefaultMigrationFiles creates default migration files if they don't exist
func (m *MigrationManager) createDefaultMigrationFiles() error {
	log.Println("Creating default migration files...")

	// This method would create the actual .sql files
	// For now, we'll just create empty directories and let user create files manually
	log.Println("Migration directories created. Please add your .sql migration files.")
	log.Println("Expected structure:")
	log.Println("  database/migrations/user_management/001_create_schema.sql")
	log.Println("  database/migrations/sensor_data/008_create_schema.sql")
	log.Println("  etc...")

	return nil
}

// CreateMigrationFile creates a new migration file template
func (m *MigrationManager) CreateMigrationFile(module, description string) error {
	// Get next version number
	nextVersion, err := m.getNextVersion()
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	// Create filename
	filename := fmt.Sprintf("%03d_%s.sql", nextVersion, strings.ReplaceAll(description, " ", "_"))
	filePath := filepath.Join(m.migrationsDir, module, filename)

	// Create file content template
	template := fmt.Sprintf(`-- Migration: %s
-- Module: %s
-- Description: %s

-- UP
-- Add your UP migration SQL here


-- DOWN
-- Add your DOWN migration SQL here (for rollback)

`, filename, module, description)

	// Write file
	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	log.Printf("Migration file created: %s", filePath)
	return nil
}

// getNextVersion returns the next available version number
func (m *MigrationManager) getNextVersion() (int, error) {
	var maxVersion int
	err := m.db.QueryRow("SELECT COALESCE(MAX(CAST(version AS INTEGER)), 0) FROM public.migrations").Scan(&maxVersion)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get max version: %w", err)
	}

	return maxVersion + 1, nil
}
