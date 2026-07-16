//go:build darwin

package updater

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	osGoos = func() string { return "darwin" }
	osGoarch = func() string {
		cmd := exec.Command("uname", "-m")
		out, err := cmd.Output()
		if err != nil {
			return "arm64"
		}
		arch := string(out)
		if arch == "x86_64\n" {
			return "amd64"
		}
		return "arm64"
	}
}

func (u *Updater) Replace(execPath, downloadedPath string) error {
	// macOS: replace .app bundle
	appDir := filepath.Dir(filepath.Dir(filepath.Dir(execPath))) // .../QuantFlow.app/Contents/MacOS/quantflow -> .../QuantFlow.app
	tmpDir, err := os.MkdirTemp(filepath.Dir(appDir), "quantflow-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Unzip downloaded archive
	cmd := exec.Command("unzip", "-q", downloadedPath, "-d", tmpDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unzip update: %w", err)
	}

	// Find .app bundle in temp
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("read temp dir: %w", err)
	}
	var newApp string
	for _, e := range entries {
		if e.IsDir() && filepath.Ext(e.Name()) == ".app" {
			newApp = filepath.Join(tmpDir, e.Name())
			break
		}
	}
	if newApp == "" {
		return fmt.Errorf("no .app bundle found in archive")
	}

	// Replace old app bundle
	backupPath := appDir + ".bak"
	os.RemoveAll(backupPath)
	if err := os.Rename(appDir, backupPath); err != nil {
		return fmt.Errorf("backup existing app: %w", err)
	}
	if err := os.Rename(newApp, appDir); err != nil {
		// Restore backup
		os.Rename(backupPath, appDir)
		return fmt.Errorf("replace app bundle: %w", err)
	}
	os.RemoveAll(backupPath)

	return nil
}

// Restart re-launches the application and exits the current process.
func Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	slog.Info("restarting application", "path", execPath)
	// Walk from binary (.../QuantFlow.app/Contents/MacOS/quantflow) to the .app bundle
	appPath := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	cmd := exec.Command("open", appPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	os.Exit(0)
	return nil // unreachable; os.Exit(0) terminates the process
}
