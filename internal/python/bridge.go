// Package python provides a gRPC-based bridge to the Python sidecar.
// The Python sidecar must be running (python -m src.server) for bridge operations to succeed.
// When Python is unavailable, bridge methods return clear errors rather than crashing.
package python

import (
	"context"
	"fmt"
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

	return &PythonBridge{
		conn:         conn,
		FactorClient: pb.NewFactorServiceClient(conn),
		LLMClient:    pb.NewLLMServiceClient(conn),
		HealthClient: pb.NewHealthServiceClient(conn),
		DataClient:      pb.NewDataServiceClient(conn),
		SentimentClient: pb.NewSentimentServiceClient(conn),
		opts:            opts,
	}, nil
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

// GetStatus returns detailed status information from the Python sidecar.
func (b *PythonBridge) GetStatus(ctx context.Context) (*pb.StatusResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()
	return b.HealthClient.GetStatus(ctx, &pb.GetStatusRequest{})
}

// Close closes the gRPC connection. Call when shutting down.
func (b *PythonBridge) Close() error {
	return b.conn.Close()
}

// RequestTimeout returns the configured request timeout.
func (b *PythonBridge) RequestTimeout() time.Duration { return b.opts.RequestTimeout }

// MaxRetries returns the configured max retry count.
func (b *PythonBridge) MaxRetries() int { return b.opts.MaxRetries }

// isTransient returns true if the gRPC error is likely transient (worth retrying).
func isTransient(err error) bool {
	// Use the canonical gRPC status code first.
	if code := status.Code(err); code != codes.Unknown {
		switch code {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
			return true
		default:
			return false
		}
	}
	// Fall back to substring matching for non-gRPC transport errors.
	s := err.Error()
	for _, marker := range []string{"connection refused", "connection reset", "EOF"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// AnalyzeChanlun calls the Python sidecar to perform 缠论 (Chanlun) analysis on a symbol.
// Returns fractals, bi segments (笔), and zhongshu blocks (中枢).
// TODO: Add gRPC service endpoints in proto and Python sidecar.
func (b *PythonBridge) AnalyzeChanlun(symbol string) (map[string]any, error) {
	// Stub: Python sidecar does not yet expose a chanlun service.
	// When the chanlun API is added to proto/ and python/src/, replace this
	// with a real gRPC call via a ChanlunClient.
	return map[string]any{
		"fractals":  []any{},
		"bi_list":   []any{},
		"zs_list":   []any{},
		"symbol":    symbol,
		"available": false,
	}, nil
}

// ComputeIndicator calculates a technical indicator via the Python sidecar.
// indicatorName is an ID like "kdj", "dmi", "atr", etc.
// params carries indicator-specific parameters (e.g. {"n": 9, "m1": 3}).
// TODO: Add gRPC service endpoints in proto and Python sidecar.
func (b *PythonBridge) ComputeIndicator(symbol string, indicatorName string, params map[string]any) (map[string]any, error) {
	// Stub: Python sidecar does not yet expose an indicator service.
	// When the indicator API is added to proto/ and python/src/, replace this
	// with a real gRPC call via a TechIndicatorClient.
	return map[string]any{
		"symbol":     symbol,
		"indicator":  indicatorName,
		"params":     params,
		"data":       []any{},
		"available":  false,
	}, nil
}
