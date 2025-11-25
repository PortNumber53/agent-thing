package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const defaultMigrationFileMode = 0o644

func maybeHandleMigrateSubcommand(cfg *Config) bool {
	if len(os.Args) < 2 || os.Args[1] != "migrate" {
		return false
	}
	if len(os.Args) < 3 {
		log.Fatalf("Usage: go run ./backend migrate <up|down|status|create>")
	}

	command := os.Args[2]

	if command == "create" {
		if len(os.Args) < 4 {
			log.Fatalf("Usage: go run ./backend migrate create <name>")
		}
		name := os.Args[3]
		timestamp := time.Now().Format("20060102150405")
		upFile := fmt.Sprintf("db/migrations/%s_%s.up.sql", timestamp, name)
		downFile := fmt.Sprintf("db/migrations/%s_%s.down.sql", timestamp, name)
		_ = os.MkdirAll(filepath.Dir(upFile), 0o755)

		if err := os.WriteFile(upFile, []byte("-- up migration here"), defaultMigrationFileMode); err != nil {
			log.Fatalf("Failed to create up migration file: %v", err)
		}
		if err := os.WriteFile(downFile, []byte("-- down migration here"), defaultMigrationFileMode); err != nil {
			log.Fatalf("Failed to create down migration file: %v", err)
		}

		fmt.Printf("Created migration files:\n%s\n%s\n", upFile, downFile)
		return true
	}

	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = cfg.XataDatabaseURL
	}
	if dsn == "" {
		log.Fatalf("DATABASE_URL or XATA_DATABASE_URL must be set for migrations")
	}

	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while migrating up: %v", err)
		}
		fmt.Println("Migrations applied successfully.")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while migrating down: %v", err)
		}
		fmt.Println("Migrations rolled back successfully.")
	case "status":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
		fmt.Printf("Version: %d, Dirty: %v\n", version, dirty)
	default:
		log.Fatalf("Unknown migrate command: %s", command)
	}

	return true
}
