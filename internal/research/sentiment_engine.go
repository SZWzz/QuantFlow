package research

import (
	"context"
	"fmt"
	"log/slog"
	"quantflow/internal/market/adapters"
	"quantflow/internal/python"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	pb "quantflow/internal/python/proto"
)

// SentimentEngine orchestrates sentiment analysis across cache, gRPC, and mock fallback.
// When textContent is empty, it auto-fetches news via the newsAdapter (if set) to provide
// real text for NLP analysis.
type SentimentEngine struct {
	bridge      *python.PythonBridge // nil when Python sidecar is unavailable
	repo        *ResearchRepo
	newsAdapter adapters.NewsAdapter // optional: auto-fetches news when textContent is empty
}

// NewSentimentEngine creates a new SentimentEngine. bridge and newsAdapter may be nil.
func NewSentimentEngine(bridge *python.PythonBridge, repo *ResearchRepo, newsAdapter adapters.NewsAdapter) *SentimentEngine {
	return &SentimentEngine{bridge: bridge, repo: repo, newsAdapter: newsAdapter}
}

// AnalyzeSentiment returns sentiment for a symbol. Cache-first, then gRPC,
// then mock fallback when Python is unavailable.
//
// When textContent is empty and a newsAdapter is configured, it auto-fetches
// recent news articles for the symbol and uses the concatenated text for NLP.
func (e *SentimentEngine) AnalyzeSentiment(ctx context.Context, symbol, textContent, textType, language string) (*SentimentOutput, error) {
	// 1. Check cache
	if e.repo != nil {
		cached, err := e.repo.GetLatestSentiment(symbol)
		if err == nil && cached != nil {
			slog.Debug("sentiment cache hit", "symbol", symbol)
			return cached, nil
		}
	}

	// 2. Auto-fetch news text if none provided
	if textContent == "" && e.newsAdapter != nil {
		articles, err := e.newsAdapter.FetchStockNews(ctx, symbol, 5)
		if err != nil {
			slog.Debug("news fetch failed, proceeding without text", "symbol", symbol, "error", err)
		} else if len(articles) > 0 {
			parts := make([]string, 0, len(articles)*2)
			for _, a := range articles {
				if a.Title != "" {
					parts = append(parts, a.Title)
				}
				if a.Content != "" {
					parts = append(parts, a.Content)
				}
			}
			textContent = strings.Join(parts, ". ")
			slog.Debug("auto-fetched news for sentiment", "symbol", symbol, "articles", len(articles), "text_len", len(textContent))
		}
	}

	// 3. Try gRPC via Python bridge
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

	// 4. Mock fallback
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
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	results := make([]*SentimentOutput, len(symbols))

	for i, sym := range symbols {
		i, sym := i, sym
		g.Go(func() error {
			output, err := e.AnalyzeSentiment(ctx, sym, "", textType, language)
			if err != nil {
				output = e.mockSentiment(sym, textType)
			}
			results[i] = output
			return nil
		})
	}
	return results, g.Wait()
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
	var confidence float64
	if len(results) > 0 {
		keywords = results[0].Keywords
		entities = results[0].Entities
		confidence = results[0].Confidence
	}
	return &SentimentOutput{
		Symbol:      resp.Symbol,
		Score:       resp.OverallScore,
		Label:       resp.OverallLabel,
		Confidence:  confidence,
		Keywords:    keywords,
		Entities:    entities,
		Source:      "",
		ComputeTime: resp.ComputeTimeMs,
	}
}
