package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"quantflow/internal/data"
	"strings"
	"time"
)

func (a *App) GetStorageStats(ctx context.Context) ([]data.TableStat, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return data.GetTableStats(a.db)
}

func (a *App) ArchiveData(ctx context.Context, source, symbol, before string) (*data.ArchiveResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return data.ArchiveData(a.db, source, symbol, before)
}

func (a *App) ExportData(ctx context.Context, table, symbol, interval, format, dateFrom, dateTo string) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	var outputPath string
	var count int64
	var err error

	switch format {
	case "csv":
		outputPath = exportFilePath(table, symbol, interval, dateFrom, dateTo, "csv")
		count, err = data.ExportCSV(a.db, table, symbol, interval, dateFrom, dateTo, outputPath)
	case "parquet":
		outputPath = exportFilePath(table, symbol, interval, dateFrom, dateTo, "parquet")
		count, err = data.ExportParquet(a.db, table, symbol, interval, dateFrom, dateTo, outputPath)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("no data to export")
	}
	return outputPath, nil
}

func (a *App) ImportData(ctx context.Context, filePath, table string) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	ext := filepath.Ext(filePath)
	switch ext {
	case ".csv":
		return data.ImportCSV(a.db, filePath, table)
	case ".parquet":
		return data.ImportParquet(a.db, filePath, table)
	default:
		return 0, fmt.Errorf("unsupported file format: %s", ext)
	}
}

func (a *App) CleanupData(ctx context.Context, table, symbol, before string, dryRun bool) (*data.CleanupResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return data.CleanupData(a.db, table, symbol, before, dryRun)
}

// SaveLayout stores layout JSON under a named key in user_config.
func (a *App) SaveLayout(ctx context.Context, name string, layoutJSON string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}
	_, err := a.db.Exec(
		"INSERT OR REPLACE INTO user_config (key, value) VALUES (?, ?)",
		"layout:"+name, layoutJSON,
	)
	if err != nil {
		return fmt.Errorf("save layout: %w", err)
	}
	return nil
}

// LoadLayout retrieves layout JSON by name from user_config.
func (a *App) LoadLayout(ctx context.Context, name string) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	var value string
	err := a.db.QueryRow(
		"SELECT value FROM user_config WHERE key = ?", "layout:"+name,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("layout %q not found", name)
	}
	if err != nil {
		return "", fmt.Errorf("load layout: %w", err)
	}
	return value, nil
}

// ListLayouts returns all saved layout names.
func (a *App) ListLayouts(ctx context.Context) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := a.db.Query("SELECT key FROM user_config WHERE key LIKE 'layout:%' ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list layouts: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		names = append(names, strings.TrimPrefix(key, "layout:"))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

// DeleteLayout removes a saved layout by name.
func (a *App) DeleteLayout(ctx context.Context, name string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	res, err := a.db.Exec("DELETE FROM user_config WHERE key = ?", "layout:"+name)
	if err != nil {
		return fmt.Errorf("delete layout: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("layout %q not found", name)
	}
	return nil
}

func exportFilePath(table, symbol, interval, dateFrom, dateTo, ext string) string {
	ts := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("%s_%s_%s_%s_%s.%s", table, symbol, interval, dateFrom, dateTo, ext)
	if symbol == "" {
		name = fmt.Sprintf("%s_%s_%s_%s.%s", table, interval, dateFrom, dateTo, ext)
	}
	return filepath.Join("data", "export", ts+"_"+name)
}
