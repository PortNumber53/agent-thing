package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	rawDSN := cfg.DatabaseURL
	source := "DATABASE_URL"
	if strings.TrimSpace(rawDSN) == "" {
		rawDSN = cfg.XataDatabaseURL
		source = "XATA_DATABASE_URL"
	}
	rawDSN = sanitizeDSN(rawDSN)
	if rawDSN == "" {
		log.Fatalf("DATABASE_URL or XATA_DATABASE_URL must be set for migrations")
	}

	logMigrationDSN(source, rawDSN)
	m, err := migrate.New("file://db/migrations", rawDSN)
	if err != nil {
		log.Fatalf("Failed to create migrate instance (source=%s): %v", source, err)
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

// sanitizeDSN trims whitespace/quotes and adds postgres:// if the scheme is missing.
func sanitizeDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	dsn = strings.Trim(dsn, `"'`)
	if dsn == "" {
		return ""
	}
	if !strings.Contains(dsn, "://") {
		// Most Jenkins credential types return raw host/user strings; assume postgres if unset.
		dsn = "postgres://" + dsn
	}
	return dsn
}

// logMigrationDSN logs a safe summary without secrets.
func logMigrationDSN(source, dsn string) {
	u, err := url.Parse(dsn)
	if err != nil {
		log.Printf("migrate using %s (unable to parse url): %s", source, err)
		return
	}
	host := u.Host
	if host == "" {
		host = "(no-host)"
	}
	log.Printf("migrate using %s scheme=%s host=%s path=%s", source, u.Scheme, host, u.Path)
}
