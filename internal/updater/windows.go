//go:build windows

package updater

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	osGoos = func() string { return "windows" }
	osGoarch = func() string {
		if os.Getenv("PROCESSOR_ARCHITECTURE") == "ARM64" {
			return "arm64"
		}
		return "amd64"
	}
}

func (u *Updater) Replace(execPath, downloadedPath string) error {
	tmpDir := filepath.Dir(downloadedPath)

	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s' -Force", downloadedPath, tmpDir))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}

	newBinary := filepath.Join(tmpDir, "quantflow.exe")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		return fmt.Errorf("quantflow.exe not found in archive")
	}

	// Use a bat script to replace after exit
	batPath := filepath.Join(tmpDir, "replace.bat")
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
copy /Y "%s" "%s"
del "%s"
start "" "%s"
`, newBinary, execPath, batPath, execPath)
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("write replace script: %w", err)
	}

	cmd = exec.Command("cmd", "/C", "start", "/B", batPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start replace script: %w", err)
	}

	os.Exit(0)
	return nil
}

func Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	slog.Info("restarting application", "path", execPath)
	cmd := exec.Command(execPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	os.Exit(0)
	return nil // unreachable; os.Exit(0) terminates the process
}
