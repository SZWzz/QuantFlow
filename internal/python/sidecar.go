package python

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SidecarProcess wraps a running Python sidecar child process.
type SidecarProcess struct {
	cmd    *exec.Cmd
	addr   string
	done   chan struct{}
}

// StartSidecar launches the Python sidecar as a subprocess if it's not already running.
// Returns a SidecarProcess (nil if already running externally) and any error.
//
// The sidecar is searched relative to the executable directory:
//
//	python/.venv/bin/python -m src.server --port <port>
func StartSidecar(ctx context.Context, pythonDir string, port int) (*SidecarProcess, error) {
	addr := fmt.Sprintf("localhost:%d", port)

	// Check if already running
	if conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		conn.Close()
		slog.Info("python sidecar already running", "addr", addr)
		return nil, nil
	}

	pythonBin := filepath.Join(pythonDir, ".venv", "bin", "python3")
	if _, err := os.Stat(pythonBin); err != nil {
		// Try system python
		pythonBin = "python3"
	}

	cmd := exec.CommandContext(ctx, pythonBin, "-m", "src.server", "--port", fmt.Sprintf("%d", port))
	cmd.Dir = pythonDir
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(pythonDir, "src"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("python sidecar: failed to start: %w", err)
	}

	sp := &SidecarProcess{
		cmd:  cmd,
		addr: addr,
		done: make(chan struct{}),
	}

	// Wait for sidecar to become ready
	go sp.waitReady(ctx)
	return sp, nil
}

func (sp *SidecarProcess) waitReady(ctx context.Context) {
	defer close(sp.done)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("tcp", sp.addr, 200*time.Millisecond); err == nil {
			conn.Close()
			slog.Info("python sidecar ready", "addr", sp.addr)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
	slog.Warn("python sidecar did not become ready within timeout", "addr", sp.addr)
}

// Wait blocks until the sidecar readiness check completes.
func (sp *SidecarProcess) Wait() {
	<-sp.done
}

// Stop gracefully terminates the sidecar process.
func (sp *SidecarProcess) Stop() {
	if sp.cmd != nil && sp.cmd.Process != nil {
		sp.cmd.Process.Signal(os.Interrupt)
		select {
		case <-sp.done:
		case <-time.After(5 * time.Second):
			sp.cmd.Process.Kill()
		}
	}
}
