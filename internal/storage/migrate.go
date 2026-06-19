package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migration represents a numbered schema migration.
type Migration struct {
	Version int
	SQL     string
}

// Run applies all pending migrations in version order.
// Already-applied migrations are skipped (idempotent).
func Run(db *sql.DB, migrations []Migration) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Sort migrations by version to ensure consistent ordering.
	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Version < sorted[j].Version })

	for _, m := range sorted {
		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_version WHERE version = ?", m.Version).Scan(&exists); err != nil {
			return fmt.Errorf("check version %d: %w", m.Version, err)
		}
		if exists > 0 {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", m.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.Version, err)
		}

		slog.Info("migration applied", "version", m.Version)
	}

	return nil
}

// BuiltinMigrations returns all embedded migrations sorted by version.
// Files in the migrations directory that do not start with a numeric
// prefix (e.g. placeholder files) are silently skipped.
func BuiltinMigrations() ([]Migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var migrations []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			// Skip files without a numeric version prefix (e.g., placeholders).
			continue
		}
		sqlContent, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		migrations = append(migrations, Migration{Version: version, SQL: string(sqlContent)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}
