package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const advisoryLockID = 2026042801

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	if err := run(ctx, db, migrationsDir); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, err := db.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID); err != nil {
			log.Printf("release migration lock: %v", err)
		}
	}()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	files, err := migrationFiles(migrationsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		filename := filepath.Base(file)
		applied, err := isApplied(ctx, db, filename)
		if err != nil {
			return err
		}
		if applied {
			log.Printf("skip %s", filename)
			continue
		}

		if err := applyMigration(ctx, db, file, filename); err != nil {
			return err
		}
		log.Printf("applied %s", filename)
	}

	return nil
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func isApplied(ctx context.Context, db *sql.DB, filename string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)
	`, filename).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", filename, err)
	}
	return exists, nil
}

func applyMigration(ctx context.Context, db *sql.DB, path, filename string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", filename, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("apply migration %s: %w", filename, err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (filename) VALUES ($1)
	`, filename); err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", filename, err)
	}

	return nil
}
