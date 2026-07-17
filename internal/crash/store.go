// internal/crash/store.go
package crash

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) Save(report *CrashReport) (string, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", fmt.Errorf("create crash dir: %w", err)
	}

	path := filepath.Join(s.dir, report.FileName())
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

func (s *Store) List() ([]CrashReport, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CrashReport{}, nil
		}
		return nil, fmt.Errorf("read crash dir: %w", err)
	}

	var reports []CrashReport
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			slog.Warn("read crash file failed, skipping", "file", e.Name(), "error", err)
			continue
		}
		var report CrashReport
		if err := json.Unmarshal(data, &report); err != nil {
			slog.Warn("parse crash file failed, skipping", "file", e.Name(), "error", err)
			continue
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (s *Store) Delete(id string) error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("read crash dir: %w", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var report CrashReport
		if json.Unmarshal(data, &report) != nil {
			continue
		}
		if report.ID == id {
			return os.Remove(filepath.Join(s.dir, e.Name()))
		}
	}
	return fmt.Errorf("report %s not found", id)
}

func (s *Store) Cleanup(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read crash dir: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(s.dir, e.Name())); err != nil {
				slog.Warn("remove old crash file failed", "file", e.Name(), "error", err)
			} else {
				removed++
			}
		}
	}
	return removed, nil
}

func (s *Store) Upload(report *CrashReport, endpoint string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload status %d", resp.StatusCode)
	}
	return nil
}

func (s *Store) Dir() string {
	return s.dir
}
