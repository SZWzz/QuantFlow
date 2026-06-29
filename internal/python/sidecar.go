package python

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "quantflow/internal/python/proto"
)

// ExpectedSidecarVersion must match the version string in python/pyproject.toml.
// The sidecar reads this at startup from importlib.metadata or falls back to this string.
const ExpectedSidecarVersion = "0.2.3"

// SidecarProcess wraps a running Python sidecar child process.
type SidecarProcess struct {
	cmd     *exec.Cmd
	addr    string
	pidFile string
	done    chan struct{}
}

// StartSidecar launches the Python sidecar as a subprocess if it's not already running.
// Returns a SidecarProcess (nil if already running with a compatible version) and any error.
//
// The sidecar is searched relative to the executable directory:
//
//	python/.venv/bin/python -m src.server --port <port>
//
// If a sidecar is already listening on the port but reports an incompatible version,
// it is killed and restarted. A PID file (.quantflow-sidecar.pid) is written to the
// binary directory so future restarts can find and manage the old process.
func StartSidecar(ctx context.Context, pythonDir string, port int) (*SidecarProcess, error) {
	addr := fmt.Sprintf("localhost:%d", port)

	// PID file goes in the binary directory (one level above pythonDir).
	binDir := filepath.Dir(pythonDir)
	pidFile := filepath.Join(binDir, ".quantflow-sidecar.pid")

	// Always kill any stale sidecar process on the port before starting fresh.
	// This prevents reusing an old Python process that has stale code loaded,
	// which can happen when the Go app exits but the Python sidecar orphan survives.
	killSidecarByPort(port)
	time.Sleep(300 * time.Millisecond)
	os.Remove(pidFile)

	// Resolve the Python binary — prefer the project's venv, fall back to system.
	pythonBin := filepath.Join(pythonDir, ".venv", "bin", "python3")
	if _, err := os.Stat(pythonBin); err != nil {
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

	// Persist the PID + version so subsequent runs can detect and restart stale processes.
	pidContent := fmt.Sprintf("%d\n%s", cmd.Process.Pid, ExpectedSidecarVersion)
	if err := os.WriteFile(pidFile, []byte(pidContent), 0644); err != nil {
		slog.Warn("failed to write sidecar PID file", "path", pidFile, "error", err)
	}

	sp := &SidecarProcess{
		cmd:     cmd,
		addr:    addr,
		pidFile: pidFile,
		done:    make(chan struct{}),
	}

	// Wait for sidecar to become ready (non-blocking).
	go sp.waitReady(ctx)
	return sp, nil
}

// getSidecarVersion connects to a running sidecar and returns its reported version.
func getSidecarVersion(ctx context.Context, addr string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return "", fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	client := pb.NewHealthServiceClient(conn)
	resp, err := client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return "", fmt.Errorf("ping: %w", err)
	}
	return resp.Version, nil
}

// killSidecarByPort finds any process listening on the given TCP port and sends SIGTERM.
func killSidecarByPort(port int) {
	// lsof -ti :<port> returns just the PID(s) listening on that port.
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
	out, err := cmd.Output()
	if err != nil {
		slog.Warn("failed to find sidecar process by port", "port", port, "error", err)
		return
	}

	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		return
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		slog.Warn("invalid PID from lsof", "output", pidStr)
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		slog.Warn("failed to find sidecar process", "pid", pid, "error", err)
		return
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		slog.Warn("failed to send SIGTERM to sidecar", "pid", pid, "error", err)
		return
	}

	slog.Info("sent SIGTERM to stale python sidecar", "pid", pid)
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

// Stop gracefully terminates the sidecar process and cleans up the PID file.
func (sp *SidecarProcess) Stop() {
	if sp.cmd != nil && sp.cmd.Process != nil {
		sp.cmd.Process.Signal(os.Interrupt)
		select {
		case <-sp.done:
		case <-time.After(5 * time.Second):
			sp.cmd.Process.Kill()
		}
	}
	if sp.pidFile != "" {
		os.Remove(sp.pidFile)
	}
}
