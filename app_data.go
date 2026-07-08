package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"quantflow/internal/data"
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

func exportFilePath(table, symbol, interval, dateFrom, dateTo, ext string) string {
	ts := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("%s_%s_%s_%s_%s.%s", table, symbol, interval, dateFrom, dateTo, ext)
	if symbol == "" {
		name = fmt.Sprintf("%s_%s_%s_%s.%s", table, interval, dateFrom, dateTo, ext)
	}
	return filepath.Join("data", "export", ts+"_"+name)
}
