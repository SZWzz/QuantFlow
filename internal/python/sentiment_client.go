package python

import (
	"context"
	"fmt"
	"time"

	pb "quantflow/internal/python/proto"
)

// AnalyzeSentiment calls the Python sidecar to analyze sentiment for a symbol.
func (b *PythonBridge) AnalyzeSentiment(ctx context.Context, symbol, textContent, textType, language string) (*pb.AnalyzeSentimentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.AnalyzeSentimentRequest{
		Symbol:      symbol,
		TextContent: textContent,
		TextType:    textType,
		Language:    language,
		MaxSources:  10,
	}

	var lastErr error
	for attempt := 0; attempt < b.opts.MaxRetries; attempt++ {
		resp, err := b.SentimentClient.AnalyzeSentiment(ctx, req)
		if err != nil {
			lastErr = err
			if isTransient(err) && attempt < b.opts.MaxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("analyze sentiment %q: %w", symbol, err)
		}

		if resp.Error != "" {
			return nil, fmt.Errorf("python sentiment error: %s", resp.Error)
		}
		return resp, nil
	}

	return nil, fmt.Errorf("analyze sentiment %q after %d retries: %w", symbol, b.opts.MaxRetries, lastErr)
}

// BatchAnalyzeSentiment analyzes sentiment for multiple symbols in one call.
func (b *PythonBridge) BatchAnalyzeSentiment(ctx context.Context, symbols []string, textType, language string) (*pb.BatchAnalyzeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.BatchAnalyzeRequest{
		Symbols:  symbols,
		TextType: textType,
		Language: language,
	}

	resp, err := b.SentimentClient.BatchAnalyzeSentiment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("batch sentiment: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("batch sentiment error: %s", resp.Error)
	}
	return resp, nil
}
