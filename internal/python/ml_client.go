package python

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "quantflow/internal/python/proto"
)

// MLClient wraps the gRPC MLService client with timeout and retry logic.
type MLClient struct {
	client pb.MLServiceClient
	bridge *PythonBridge
}

// NewMLClient creates a new ML client over the bridge connection.
func NewMLClient(bridge *PythonBridge) *MLClient {
	return &MLClient{
		client: pb.NewMLServiceClient(bridge.conn),
		bridge: bridge,
	}
}

// Train sends a training request to the Python sidecar with retry on transient errors.
func (c *MLClient) Train(ctx context.Context, req *pb.TrainRequest) (*pb.TrainResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= c.bridge.MaxRetries(); attempt++ {
		resp, err := c.client.Train(ctx, req)
		if err != nil {
			if isTransient(err) && attempt < c.bridge.MaxRetries() {
				lastErr = err
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("train: %w", err)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("train: max retries exceeded: %w", lastErr)
}

// Predict sends a prediction request to the Python sidecar.
func (c *MLClient) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	resp, err := c.client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("predict: %w", err)
	}
	return resp, nil
}

// Evaluate sends an evaluation request to the Python sidecar.
func (c *MLClient) Evaluate(ctx context.Context, req *pb.EvaluateRequest) (*pb.EvaluateResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	resp, err := c.client.Evaluate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	return resp, nil
}

// AlphaMining sends a factor mining request (Phase 10.2).
func (c *MLClient) AlphaMining(ctx context.Context, req *pb.AlphaMiningRequest) (*pb.AlphaMiningResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	resp, err := c.client.AlphaMining(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("alpha_mining: %w", err)
	}
	return resp, nil
}

// RLTrain starts RL training and returns a channel that receives progress updates.
func (c *MLClient) RLTrain(ctx context.Context, req *pb.RLTrainRequest) (<-chan *pb.RLTrainUpdate, error) {
	stream, err := c.client.RLTrain(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rl_train: %w", err)
	}

	ch := make(chan *pb.RLTrainUpdate, 10)
	go func() {
		defer close(ch)
		for {
			update, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			select {
			case ch <- update:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// RLPredict sends an RL inference request (Phase 10.3).
func (c *MLClient) RLPredict(ctx context.Context, req *pb.RLPredictRequest) (*pb.RLPredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	resp, err := c.client.RLPredict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rl_predict: %w", err)
	}
	return resp, nil
}

// RiskModel sends a risk modeling request (Phase 10.4).
func (c *MLClient) RiskModel(ctx context.Context, req *pb.RiskModelRequest) (*pb.RiskModelResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	resp, err := c.client.RiskModel(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("risk_model: %w", err)
	}
	return resp, nil
}
