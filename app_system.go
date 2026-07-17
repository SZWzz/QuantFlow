package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"time"

	"quantflow/internal/crash"
	"quantflow/internal/updater"
)

// GetSystemStats returns runtime statistics for the system monitor panel.
func (a *App) GetSystemStats(ctx context.Context) map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]interface{}{
		"goroutines":     runtime.NumGoroutine(),
		"mem_alloc_mb":   int(m.Alloc / 1024 / 1024),
		"mem_sys_mb":     int(m.Sys / 1024 / 1024),
		"num_gc":         int(m.NumGC),
		"go_version":     runtime.Version(),
		"uptime_seconds": int(time.Since(startTime).Seconds()),
	}
}

// GetConfig returns the current application configuration (non-sensitive).
func (a *App) GetConfig() map[string]interface{} {
	// Frontend settings (theme/language/density) live in localStorage.
	// Only expose non-sensitive config here. api_keys are NEVER sent to frontend (audit fix C4).
	return map[string]interface{}{
		"version":  a.cfg.Version,
		"logLevel": a.cfg.LogLevel,
		"dbPath":   a.resolvedDBPath,
	}
}

// UpdateConfig merges partial config into the current config and persists to config.yaml.
func (a *App) UpdateConfig(ctx context.Context, patch map[string]interface{}) error {
	if keys, ok := patch["api_keys"].(map[string]interface{}); ok {
		for k, v := range keys {
			if s, ok := v.(string); ok {
				a.cfg.APIKeys[k] = s
			}
		}
	}
	return a.cfg.Save()
}

// ── Connection Status ──────────────────────────────────────────────────

// ConnectionStatus represents the live status of data sources, brokers, and Python sidecar.
type ConnectionStatus struct {
	Markets map[string]string `json:"markets"`
	Brokers map[string]string `json:"brokers"`
	Python  string            `json:"python"`
}

// GetConnectionStatus returns the live connection status for the StatusBar.
func (a *App) GetConnectionStatus() ConnectionStatus {
	status := ConnectionStatus{
		Markets: map[string]string{
			"A股":  "未配置",
			"港股": "未配置",
			"美股": "未配置",
			"加密": "未配置",
		},
		Brokers: make(map[string]string),
		Python:  "未连接",
	}

	// Market adapter status
	if a.marketReg != nil {
		// Check presence of adapters for each market
		haveCN := false
		haveHK := false
		haveUS := false
		haveCrypto := false
		for _, name := range a.marketReg.List() {
			adp := a.marketReg.Get(name)
			if adp == nil {
				continue
			}
			for _, mkt := range adp.Markets() {
				switch mkt {
				case "CN":
					haveCN = true
				case "HK":
					haveHK = true
				case "US":
					haveUS = true
				case "CRYPTO":
					haveCrypto = true
				}
			}
		}
		if haveCN {
			status.Markets["A股"] = "已配置"
		}
		if haveHK {
			status.Markets["港股"] = "已配置"
		}
		if haveUS {
			status.Markets["美股"] = "已配置"
		}
		if haveCrypto {
			status.Markets["加密"] = "已配置"
		}
	}

	// Broker status — use IsConnected() from the Broker interface
	if a.brokers != nil {
		for name, broker := range a.brokers {
			if broker.IsConnected() {
				status.Brokers[name] = "已连接"
			} else {
				status.Brokers[name] = "未连接"
			}
		}
	}

	// Python sidecar — running if initialized
	if a.sidecar != nil {
		status.Python = "运行中"
	}

	return status
}

func marketLabel(mkt string) string {
	switch mkt {
	case "CN":
		return "A股"
	case "HK":
		return "港股"
	case "US":
		return "美股"
	case "CRYPTO":
		return "加密"
	default:
		return mkt
	}
}

// buildVersion is set at build time via ldflags.
// Default matches the frontend package.json version (updated per CLAUDE.md rule 3).
var buildVersion = "2026.7.17"

// GetVersion returns the application version.
// Prefers the build-time version; falls back to config, then the default above.
func (a *App) GetVersion() string {
	if buildVersion != "" && buildVersion != "0.0.0" {
		return buildVersion
	}
	if a.cfg != nil && a.cfg.Version != "" {
		return a.cfg.Version
	}
	return buildVersion
}

const updaterOwner = "SZWzz"
const updaterRepo = "QuantFlow"

// CheckUpdate checks for a new version. Returns update info or nil if up-to-date.
func (a *App) CheckUpdate() *updater.UpdateInfo {
	if a.cfg == nil {
		return &updater.UpdateInfo{HasUpdate: false}
	}

	u := updater.New(updaterOwner, updaterRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := u.Check(ctx, a.cfg.Version)
	if err != nil {
		slog.Warn("check update failed", "error", err)
		return &updater.UpdateInfo{HasUpdate: false}
	}
	return info
}

// ApplyUpdate downloads, verifies, and applies an update.
func (a *App) ApplyUpdate(assetURL, checksum string) error {
	u := updater.New(updaterOwner, updaterRepo)

	downloadedPath, err := u.Download(context.Background(), assetURL, os.TempDir(), nil)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if checksum != "" {
		if err := u.Verify(downloadedPath, checksum); err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	if err := u.Replace(execPath, downloadedPath); err != nil {
		return fmt.Errorf("replace failed: %w", err)
	}

	return updater.Restart()
}

// GetUpdateInterval returns the current update check interval setting.
func (a *App) GetUpdateInterval() string {
	if a.cfg == nil {
		return "daily"
	}
	if a.cfg.UpdateCheckInterval == "" {
		return "daily"
	}
	return a.cfg.UpdateCheckInterval
}

// SetUpdateInterval sets the update check interval and persists config.
func (a *App) SetUpdateInterval(interval string) error {
	if a.cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	switch interval {
	case "always", "daily", "never":
		a.cfg.UpdateCheckInterval = interval
		return a.cfg.Save()
	default:
		return fmt.Errorf("invalid interval: %s (must be always/daily/never)", interval)
	}
}

// OpenURL opens a URL in the system's default browser.
func (a *App) OpenURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("open", url).Start()
	}
}

// ── Crash Reports ──────────────────────────────────────────────────────

// defaultCrashEndpoint is used for opt-in crash report uploads when the user
// has not configured a custom "crash_endpoint" in the config API keys.
const defaultCrashEndpoint = "https://hooks.quantflow.app/crashes"

// ListCrashReports returns all saved crash reports from previous sessions.
// Reports contain no API keys or position details (see internal/crash docs).
func (a *App) ListCrashReports() []crash.CrashReport {
	if a.crashStore == nil {
		return nil
	}
	reports, err := a.crashStore.List()
	if err != nil {
		slog.Warn("list crash reports", "error", err)
		return nil
	}
	return reports
}

// DeleteCrashReport deletes a crash report by ID.
func (a *App) DeleteCrashReport(id string) error {
	if a.crashStore == nil {
		return fmt.Errorf("crash store not initialized")
	}
	return a.crashStore.Delete(id)
}

// UploadCrashReport uploads a crash report to the configured endpoint.
// Upload is strictly opt-in: it only happens when the frontend calls this
// method after explicit user consent (crash dialog checkbox / history panel).
func (a *App) UploadCrashReport(id string) error {
	if a.crashStore == nil {
		return fmt.Errorf("crash store not initialized")
	}
	reports, err := a.crashStore.List()
	if err != nil {
		return fmt.Errorf("list reports: %w", err)
	}
	for i := range reports {
		if reports[i].ID == id {
			endpoint := defaultCrashEndpoint
			if a.cfg != nil {
				if custom := a.cfg.GetAPIKey("crash_endpoint"); custom != "" {
					endpoint = custom
				}
			}
			return a.crashStore.Upload(&reports[i], endpoint)
		}
	}
	return fmt.Errorf("report %s not found", id)
}

// GetCrashDir returns the directory where crash reports are stored, for
// display in the crash recovery dialog.
func (a *App) GetCrashDir() string {
	if a.crashStore == nil {
		return crash.CrashDir()
	}
	return a.crashStore.Dir()
}
