package database

import (
	"database/sql"
	"fmt"
	"log"
	"user-management/config"

	_ "github.com/lib/pq"
)

// DB holds database connection
type DB struct {
	*sql.DB
}

// Schema name constants
const (
	UserManagementSchema = "user_management"
)

// NewConnection creates a new database connection
func NewConnection(cfg *config.DatabaseConfig) (*DB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database %s:%d", cfg.Host, cfg.Port)

	return &DB{db}, nil
}

// MustConnect creates database connection or panics
func MustConnect(cfg *config.DatabaseConfig) *DB {
	db, err := NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

// Close closes database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// RunMigrations runs all database migrations
func (db *DB) RunMigrations() error {
	migrationManager := NewMigrationManager(db.DB)
	return migrationManager.RunMigrations()
}
