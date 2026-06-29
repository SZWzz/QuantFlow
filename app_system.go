package main

import (
	"context"
	"os/exec"
	"runtime"
	"time"
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
		"dbPath":   a.cfg.DBPath,
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

// GetVersion returns the application version.
func (a *App) GetVersion() string {
	if a.cfg == nil {
		return "unknown"
	}
	return a.cfg.Version
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
