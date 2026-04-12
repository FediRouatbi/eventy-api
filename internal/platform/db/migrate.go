package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func MigrateUp(ctx context.Context, db *sql.DB, source string, steps int) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	migrationFiles, err := listMigrationFiles(source, ".up.sql")
	if err != nil {
		return err
	}

	appliedCount := 0
	for _, migrationFile := range migrationFiles {
		applied, err := migrationApplied(ctx, db, migrationFile.Name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := applyMigration(ctx, db, migrationFile); err != nil {
			return err
		}

		appliedCount++
		if steps > 0 && appliedCount >= steps {
			break
		}
	}

	return nil
}

type migrationFile struct {
	Name string
	Path string
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)
`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	return nil
}

func listMigrationFiles(source string, suffix string) ([]migrationFile, error) {
	migrationsDir, err := resolveMigrationSource(source)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}

		files = append(files, migrationFile{
			Name: entry.Name(),
			Path: filepath.Join(migrationsDir, entry.Name()),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func resolveMigrationSource(source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", fmt.Errorf("migration source is required")
	}

	if strings.HasPrefix(trimmed, "file://") {
		trimmed = strings.TrimPrefix(trimmed, "file://")
	}

	if trimmed == "" {
		return "", fmt.Errorf("migration source is required")
	}

	return filepath.Clean(trimmed), nil
}

func applyMigration(ctx context.Context, db *sql.DB, migration migrationFile) error {
	contents, err := os.ReadFile(migration.Path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", migration.Name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Name, err)
	}

	if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply migration %s: %w", migration.Name, err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO schema_migrations (version)
VALUES (?)
`, migration.Name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", migration.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Name, err)
	}

	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM schema_migrations
    WHERE version = ?
)
`, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}

	return exists, nil
}

func ParseMigrationSteps(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}

	steps, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid migration step count: %w", err)
	}
	if steps < 0 {
		return 0, fmt.Errorf("migration step count must be zero or greater")
	}

	return steps, nil
}
