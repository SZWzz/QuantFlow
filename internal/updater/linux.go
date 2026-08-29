//go:build linux

package updater

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	osGoos = func() string { return "linux" }
	osGoarch = func() string {
		cmd := exec.Command("uname", "-m")
		out, err := cmd.Output()
		if err != nil {
			return "amd64"
		}
		arch := string(out)
		if arch == "aarch64\n" || arch == "arm64\n" {
			return "arm64"
		}
		return "amd64"
	}
}

func (u *Updater) Replace(execPath, downloadedPath string) error {
	tmpDir, err := os.MkdirTemp(filepath.Dir(execPath), "quantflow-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("tar", "-xzf", downloadedPath, "-C", tmpDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	newBinary := filepath.Join(tmpDir, "quantflow")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		return fmt.Errorf("binary not found in archive")
	}

	if err := os.Rename(newBinary, execPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	if err := os.Chmod(execPath, 0o755); err != nil {
		return fmt.Errorf("chmod binary: %w", err)
	}

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
