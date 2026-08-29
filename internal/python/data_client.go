package python

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	pb "quantflow/internal/python/proto"
)

// DataClient wraps the gRPC DataService client with timeout and retry logic.
// Used by data source adapters (mootdx, etc.) that need Python-side libraries.
type DataClient struct {
	client pb.DataServiceClient
	bridge *PythonBridge
}

// NewDataClient creates a new data client over the bridge connection.
func NewDataClient(bridge *PythonBridge) *DataClient {
	return &DataClient{
		client: pb.NewDataServiceClient(bridge.conn),
		bridge: bridge,
	}
}

// FetchData sends a data fetch request to the Python sidecar.
func (c *DataClient) FetchData(ctx context.Context, req *pb.FetchDataRequest) (*pb.FetchDataResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.bridge.RequestTimeout())
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= c.bridge.MaxRetries(); attempt++ {
		resp, err := c.client.FetchData(ctx, req)
		if err != nil {
			if isTransient(err) && attempt < c.bridge.MaxRetries() {
				lastErr = err
				// Linear backoff with jitter: baseDelay * (attempt+1) ± 25%.
				baseDelay := time.Duration(attempt+1) * 100 * time.Millisecond
				jitter := time.Duration(rand.Int63n(int64(baseDelay) / 2)) //nolint:gosec // 重试抖动非安全用途，math/rand 足够
				sleepDuration := baseDelay + jitter
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("fetch_data: %w", ctx.Err())
				case <-time.After(sleepDuration):
				}
				continue
			}
			return nil, fmt.Errorf("fetch_data: %w", err)
		}
		if resp.Error != "" {
			return resp, fmt.Errorf("fetch_data: %s", resp.Error)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("fetch_data: max retries exceeded: %w", lastErr)
}
