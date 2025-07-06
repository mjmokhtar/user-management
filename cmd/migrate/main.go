package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"user-management/config"
	"user-management/database"
)

func main() {
	var (
		configPath = flag.String("config", "app.toml", "Path to config file")
		action     = flag.String("action", "up", "Migration action: up, down, status, reset")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migration manager
	migrationManager := database.NewMigrationManager(db.DB)

	// Execute action
	switch *action {
	case "up":
		if err := migrationManager.RunMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("✅ Migrations completed successfully")

	case "down":
		if err := migrationManager.Rollback(); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("✅ Migration rolled back successfully")

	case "status":
		if err := showMigrationStatus(db); err != nil {
			log.Fatalf("Failed to show migration status: %v", err)
		}

	case "reset":
		if err := resetDatabase(db, cfg); err != nil {
			log.Fatalf("Failed to reset database: %v", err)
		}
		fmt.Println("✅ Database reset successfully")

	default:
		fmt.Printf("Unknown action: %s\n", *action)
		fmt.Println("Available actions: up, down, status, reset")
		os.Exit(1)
	}
}

// showMigrationStatus displays current migration status
func showMigrationStatus(db *database.DB) error {
	fmt.Println("📊 Migration Status:")
	fmt.Println("==================")

	migrationManager := database.NewMigrationManager(db.DB)
	status, err := migrationManager.GetMigrationStatus()
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	if len(status) == 0 {
		fmt.Println("No migrations executed yet")
		return nil
	}

	fmt.Printf("%-8s %-15s %-40s %-20s\n", "Version", "Module", "Description", "Executed At")
	fmt.Println(strings.Repeat("-", 85))

	for _, migration := range status {
		fmt.Printf("%-8s %-15s %-40s %-20s\n",
			migration["version"],
			migration["module"],
			migration["description"],
			migration["executed_at"].(string)[:19])
	}

	fmt.Printf("\nTotal: %d migrations executed\n", len(status))
	return nil
}

// resetDatabase drops all tables and re-runs migrations
func resetDatabase(db *database.DB, cfg *config.Config) error {
	// Safety check for production environment
	if cfg.App.Environment == "production" {
		fmt.Println("❌ Reset operation is disabled in production environment")
		return fmt.Errorf("reset not allowed in production")
	}

	fmt.Println("⚠️  WARNING: This will completely reset the database!")
	fmt.Println("   - All schemas will be dropped")
	fmt.Println("   - All data will be permanently lost")
	fmt.Println("   - Migration history will be cleared")
	fmt.Printf("   - Database: %s\n", cfg.Database.DBName)
	fmt.Println()

	// Double confirmation for safety
	fmt.Print("Type 'RESET' to confirm (case sensitive): ")
	var response string
	fmt.Scanln(&response)

	if response != "RESET" {
		fmt.Println("Reset cancelled - confirmation failed")
		return nil
	}

	fmt.Print("Are you absolutely sure? Type 'YES' to proceed: ")
	fmt.Scanln(&response)

	if response != "YES" {
		fmt.Println("Reset cancelled")
		return nil
	}

	fmt.Println("🔄 Resetting database...")

	// Drop all schemas cascade (removes all tables)
	schemas := []string{
		"user_management", // database.UserManagementSchema
		"sensor_data",     // database.SensorDataSchema
	}

	for _, schema := range schemas {
		if _, err := db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema)); err != nil {
			return fmt.Errorf("failed to drop schema %s: %w", schema, err)
		}
		fmt.Printf("   ✓ Dropped schema: %s\n", schema)
	}

	// Drop migrations table
	if _, err := db.Exec("DROP TABLE IF EXISTS public.migrations"); err != nil {
		return fmt.Errorf("failed to drop migrations table: %w", err)
	}
	fmt.Println("   ✓ Dropped migrations table")

	// Re-run migrations
	fmt.Println("🔄 Re-running migrations...")
	migrationManager := database.NewMigrationManager(db.DB)
	if err := migrationManager.RunMigrations(); err != nil {
		return fmt.Errorf("failed to re-run migrations: %w", err)
	}

	return nil
}
