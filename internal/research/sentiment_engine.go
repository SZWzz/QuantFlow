package research

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// SentimentEngine orchestrates sentiment analysis across cache, gRPC, and mock fallback.
type SentimentEngine struct {
	bridge *python.PythonBridge // nil when Python sidecar is unavailable
	repo   *ResearchRepo
}

// NewSentimentEngine creates a new SentimentEngine. bridge may be nil.
func NewSentimentEngine(bridge *python.PythonBridge, repo *ResearchRepo) *SentimentEngine {
	return &SentimentEngine{bridge: bridge, repo: repo}
}

// AnalyzeSentiment returns sentiment for a symbol. Cache-first, then gRPC,
// then mock fallback when Python is unavailable.
func (e *SentimentEngine) AnalyzeSentiment(ctx context.Context, symbol, textContent, textType, language string) (*SentimentOutput, error) {
	// 1. Check cache
	if e.repo != nil {
		cached, err := e.repo.GetLatestSentiment(symbol)
		if err == nil && cached != nil {
			slog.Debug("sentiment cache hit", "symbol", symbol)
			return cached, nil
		}
	}

	// 2. Try gRPC via Python bridge
	if e.bridge != nil {
		resp, err := e.bridge.AnalyzeSentiment(ctx, symbol, textContent, textType, language)
		if err != nil {
			slog.Warn("sentiment gRPC failed, using mock", "symbol", symbol, "error", err)
		} else {
			output := pbToSentimentOutput(resp)
			if e.repo != nil {
				if err := e.repo.SaveSentiment(output); err != nil {
					slog.Warn("failed to cache sentiment", "symbol", symbol, "error", err)
				}
			}
			return output, nil
		}
	}

	// 3. Mock fallback
	return e.mockSentiment(symbol, textType), nil
}

// GetSentimentHistory returns historical sentiment records.
func (e *SentimentEngine) GetSentimentHistory(ctx context.Context, symbol string, days int) ([]SentimentOutput, error) {
	if e.repo == nil {
		return nil, fmt.Errorf("sentiment engine: repo not initialized")
	}

	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	results, err := e.repo.GetSentimentHistory(symbol, since)
	if err != nil {
		return nil, fmt.Errorf("sentiment history: %w", err)
	}
	if len(results) == 0 {
		output := e.mockSentiment(symbol, "news")
		output.Score = 0.0
		return []SentimentOutput{*output}, nil
	}
	return results, nil
}

// BatchAnalyze analyzes sentiment for multiple symbols concurrently.
func (e *SentimentEngine) BatchAnalyze(ctx context.Context, symbols []string, textType, language string) ([]*SentimentOutput, error) {
	results := make([]*SentimentOutput, len(symbols))
	for i, sym := range symbols {
		output, err := e.AnalyzeSentiment(ctx, sym, "", textType, language)
		if err != nil {
			output = e.mockSentiment(sym, textType)
		}
		results[i] = output
	}
	return results, nil
}

// IsBridgeAvailable returns whether the Python sidecar is connected.
func (e *SentimentEngine) IsBridgeAvailable() bool {
	return e.bridge != nil
}

// mockSentiment returns neutral mock sentiment data.
func (e *SentimentEngine) mockSentiment(symbol, textType string) *SentimentOutput {
	return &SentimentOutput{
		Symbol:      symbol,
		Score:       0.0,
		Label:       "neutral",
		Confidence:  0.0,
		Keywords:    []string{"mock_data"},
		Entities:    []string{},
		Source:      textType,
		ComputeTime: 0,
	}
}

// pbToSentimentOutput converts a protobuf response to the Go domain type.
func pbToSentimentOutput(resp *pb.AnalyzeSentimentResponse) *SentimentOutput {
	results := resp.Results
	var keywords, entities []string
	if len(results) > 0 {
		keywords = results[0].Keywords
		entities = results[0].Entities
	}
	return &SentimentOutput{
		Symbol:      resp.Symbol,
		Score:       resp.OverallScore,
		Label:       resp.OverallLabel,
		Confidence:  0.7,
		Keywords:    keywords,
		Entities:    entities,
		Source:      "",
		ComputeTime: resp.ComputeTimeMs,
	}
}
