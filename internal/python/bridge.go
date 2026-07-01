// Package python provides a gRPC-based bridge to the Python sidecar.
// The Python sidecar must be running (python -m src.server) for bridge operations to succeed.
// When Python is unavailable, bridge methods return clear errors rather than crashing.
package python

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "quantflow/internal/python/proto"
)

// BridgeOptions configures the PythonBridge connection.
type BridgeOptions struct {
	Address        string
	DialTimeout    time.Duration
	RequestTimeout time.Duration
	MaxRetries     int
	PythonDir      string // path to python/ directory (for subprocess calls like chanlun)
}

// DefaultOptions returns sensible defaults for local development.
func DefaultOptions() BridgeOptions {
	return BridgeOptions{
		Address:        "localhost:50051",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
	}
}

// PythonBridge manages the gRPC connection to the Python sidecar.
type PythonBridge struct {
	conn         *grpc.ClientConn
	FactorClient pb.FactorServiceClient
	LLMClient    pb.LLMServiceClient
	HealthClient pb.HealthServiceClient
	DataClient      pb.DataServiceClient
	SentimentClient pb.SentimentServiceClient
	opts            BridgeOptions
	pythonDir       string
}

// NewPythonBridge dials the Python sidecar and returns a bridge.
// Returns an error if the sidecar is unreachable or the connection fails.
func NewPythonBridge(opts BridgeOptions) (*PythonBridge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opts.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, opts.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("python bridge: dial %s: %w", opts.Address, err)
	}

	b := &PythonBridge{
		conn:         conn,
		FactorClient: pb.NewFactorServiceClient(conn),
		LLMClient:    pb.NewLLMServiceClient(conn),
		HealthClient: pb.NewHealthServiceClient(conn),
		DataClient:      pb.NewDataServiceClient(conn),
		SentimentClient: pb.NewSentimentServiceClient(conn),
		opts:            opts,
	}

	b.pythonDir = opts.PythonDir
	if b.pythonDir == "" {
		// Fall back to executable-relative path.
		if execPath, err := os.Executable(); err == nil {
			b.pythonDir = filepath.Join(filepath.Dir(execPath), "python")
		}
	}
	return b, nil
}

// Ping checks whether the Python sidecar is responsive.
func (b *PythonBridge) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	resp, err := b.HealthClient.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	if !resp.Healthy {
		return fmt.Errorf("python sidecar reports unhealthy")
	}
	return nil
}

// IsHealthy returns true if the Python sidecar is responding to pings.
func (b *PythonBridge) IsHealthy(ctx context.Context) bool {
	return b.Ping(ctx) == nil
}

// GetStatus returns detailed status from the Python sidecar.
func (b *PythonBridge) GetStatus(ctx context.Context) (*pb.StatusResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()
	return b.HealthClient.GetStatus(ctx, &pb.GetStatusRequest{})
}

// PythonDir returns the path to the Python sidecar directory.
func (b *PythonBridge) PythonDir() string { return b.pythonDir }

// Close closes the gRPC connection to the Python sidecar.
func (b *PythonBridge) Close() error {
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

// runPython executes a Python module as a subprocess and unmarshals its JSON stdout.
func (b *PythonBridge) runPython(module string, args ...string) (map[string]any, error) {
	pythonDir := b.pythonDir
	if pythonDir == "" {
		return nil, fmt.Errorf("python directory unknown")
	}

	pythonBin := filepath.Join(pythonDir, ".venv", "bin", "python3")
	if _, err := os.Stat(pythonBin); err != nil {
		pythonBin = "python3"
	}

	cmdArgs := append([]string{"-m", module}, args...)
	cmd := exec.Command(pythonBin, cmdArgs...)
	cmd.Dir = pythonDir
	cmd.Env = append(os.Environ(), "PYTHONPATH="+pythonDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("python subprocess start %s: %w", module, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, fmt.Errorf("python subprocess %s timed out", module)
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("python subprocess %s failed: %w\nstderr: %s", module, err, stderr.String())
		}
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("python subprocess %s: json decode: %w\nstdout: %s", module, err, stdout.String())
	}
	return result, nil
}

// RequestTimeout returns the configured request timeout duration.
func (b *PythonBridge) RequestTimeout() time.Duration { return b.opts.RequestTimeout }

// MaxRetries returns the configured maximum retry count.
func (b *PythonBridge) MaxRetries() int { return b.opts.MaxRetries }

// isTransient returns true if the gRPC error is likely transient (e.g., connection reset).
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	s := status.Convert(err).Message()
	if status.Code(err) == codes.Unavailable {
		return true
	}
	for _, marker := range []string{"connection refused", "connection reset", "EOF"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// AnalyzeChanlun calls the Python sidecar to perform 缠论 (Chanlun) analysis on a symbol.
func (b *PythonBridge) AnalyzeChanlun(symbol string) (map[string]any, error) {
	if b.pythonDir == "" {
		return map[string]any{
			"fractals": []any{}, "bi_list": []any{}, "zs_list": []any{},
			"symbol": symbol, "available": false,
		}, nil
	}
	result, err := b.runPython("src.data.fincept.chanlun", symbol)
	if err != nil {
		return map[string]any{
			"fractals": []any{}, "bi_list": []any{}, "zs_list": []any{},
			"symbol": symbol, "available": false, "error": err.Error(),
		}, nil
	}
	return result, nil
}

// ComputeIndicator calculates a technical indicator via the Python sidecar.
func (b *PythonBridge) ComputeIndicator(symbol string, indicatorName string, params map[string]any) (map[string]any, error) {
	if b.pythonDir == "" {
		return map[string]any{
			"symbol": symbol, "indicator": indicatorName, "params": params,
			"data": []any{}, "available": false,
		}, nil
	}

	paramsJSON := "{}"
	if params != nil {
		if p, err := json.Marshal(params); err == nil {
			paramsJSON = string(p)
		}
	}

	result, err := b.runPython("src.data.fincept.indicators", symbol, indicatorName, paramsJSON)
	if err != nil {
		return map[string]any{
			"symbol": symbol, "indicator": indicatorName, "params": params,
			"data": []any{}, "available": false, "error": err.Error(),
		}, nil
	}
	return result, nil
}

// ScanStocks runs a stock scanning strategy and returns ranked results.
func (b *PythonBridge) ScanStocks(strategyName string, topN int) (map[string]any, error) {
	if b.pythonDir == "" {
		return map[string]any{
			"strategy": strategyName, "results": []any{}, "scanned": 0,
		}, nil
	}
	topNStr := fmt.Sprintf("%d", topN)
	if topN <= 0 {
		topNStr = "50"
	}
	result, err := b.runPython("src.data.fincept.scanner", strategyName, topNStr)
	if err != nil {
		return map[string]any{
			"strategy": strategyName, "results": []any{}, "scanned": 0, "error": err.Error(),
		}, nil
	}
	return result, nil
}
