package python

import (
	"context"
	"fmt"
	"time"

	pb "quantflow/internal/python/proto"
)

// FactorResult holds the output of a single factor computation for one symbol.
type FactorResult struct {
	Symbol string
	Dates  []string
	Values []float64
}

// ComputeFactor calls the Python sidecar to compute a single factor.
// It retries on transient gRPC errors up to MaxRetries times.
func (b *PythonBridge) ComputeFactor(ctx context.Context, factorName string, symbols []string,
	startDate, endDate string, params map[string]string, ohlcvData []byte,
) ([]FactorResult, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.ComputeFactorRequest{
		FactorName: factorName,
		Symbols:    symbols,
		StartDate:  startDate,
		EndDate:    endDate,
		Params:     params,
		OhlcvData:  ohlcvData,
	}

	var lastErr error
	for attempt := 0; attempt < b.opts.MaxRetries; attempt++ {
		resp, err := b.FactorClient.ComputeFactor(ctx, req)
		if err != nil {
			lastErr = err
			if isTransient(err) && attempt < b.opts.MaxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("compute factor %q: %w", factorName, err)
		}

		if resp.Error != "" {
			return nil, fmt.Errorf("python factor error: %s", resp.Error)
		}

		results := make([]FactorResult, len(resp.Results))
		for i, r := range resp.Results {
			results[i] = FactorResult{
				Symbol: r.Symbol,
				Dates:  r.Dates,
				Values: r.Values,
			}
		}
		return results, nil
	}

	return nil, fmt.Errorf("compute factor %q after %d retries: %w", factorName, b.opts.MaxRetries, lastErr)
}

// ComputeFactorBatch computes multiple factors in a single gRPC call.
func (b *PythonBridge) ComputeFactorBatch(ctx context.Context, factorNames []string, symbols []string,
	startDate, endDate string, params map[string]string, ohlcvData []byte,
) ([]FactorResult, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.ComputeFactorBatchRequest{
		FactorNames: factorNames,
		Symbols:     symbols,
		StartDate:   startDate,
		EndDate:     endDate,
		Params:      params,
		OhlcvData:   ohlcvData,
	}

	resp, err := b.FactorClient.ComputeFactorBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("compute factor batch: %w", err)
	}

	var allResults []FactorResult
	for _, fr := range resp.FactorResponses {
		if fr.Error != "" {
			return nil, fmt.Errorf("factor %q error: %s", fr.FactorName, fr.Error)
		}
		for _, r := range fr.Results {
			allResults = append(allResults, FactorResult{
				Symbol: r.Symbol,
				Dates:  r.Dates,
				Values: r.Values,
			})
		}
	}

	return allResults, nil
}

// ListFactors returns metadata about all registered factors from the Python sidecar.
func (b *PythonBridge) ListFactors(ctx context.Context) ([]*pb.FactorMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	resp, err := b.FactorClient.ListFactors(ctx, &pb.ListFactorsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list factors: %w", err)
	}
	return resp.Factors, nil
}
