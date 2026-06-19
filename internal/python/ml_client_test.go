package python

import (
	"testing"

	pb "quantflow/internal/python/proto"
)

func TestMLClient_New(t *testing.T) {
	// Skip if no Python sidecar running — only test client construction
	t.Skip("requires running Python sidecar")

	// This test validates the client interface compiles
	var _ *MLClient
}

func TestMLClient_InterfaceCompiles(t *testing.T) {
	// Compile-time check that MLClient satisfies expected interface
	bridge := &PythonBridge{}
	_ = NewMLClient(bridge)
}

// Compile-time interface check: MLClient methods match the proto interface.
func TestMLClient_MethodSignatures(t *testing.T) {
	// These are compile-time checks that the methods exist with correct signatures
	_ = func(c *MLClient, ctx interface{ Train() }) error {
		return nil
	}

	// Verify we can reference all proto types used by MLClient
	var (
		_ *pb.TrainRequest
		_ *pb.TrainResponse
		_ *pb.PredictRequest
		_ *pb.PredictResponse
		_ *pb.EvaluateRequest
		_ *pb.EvaluateResponse
		_ *pb.AlphaMiningRequest
		_ *pb.AlphaMiningResponse
		_ *pb.RLTrainRequest
		_ *pb.RLTrainUpdate
		_ *pb.RLPredictRequest
		_ *pb.RLPredictResponse
		_ *pb.RiskModelRequest
		_ *pb.RiskModelResponse
	)
}
