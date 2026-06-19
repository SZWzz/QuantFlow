package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EastMoneyGlobalNewsAdapter fetches 7×24 rolling financial news from EastMoney.
// Implements GlobalNewsAdapter. Based on a-stock-data SKILL §5.3 (np-weblist.eastmoney.com).
//
// This is the replacement for the deprecated cls.cn (财联社) fast-news API.
type EastMoneyGlobalNewsAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
}

// Compile-time interface checks.
var _ NewsAdapter = (*EastMoneyGlobalNewsAdapter)(nil)
var _ GlobalNewsAdapter = (*EastMoneyGlobalNewsAdapter)(nil)

// NewEastMoneyGlobalNewsAdapter creates a new global news adapter.
func NewEastMoneyGlobalNewsAdapter() *EastMoneyGlobalNewsAdapter {
	return &EastMoneyGlobalNewsAdapter{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneyGlobalNewsAdapter) Name() string { return "eastmoney_global_news" }

func (a *EastMoneyGlobalNewsAdapter) IsAvailable(ctx context.Context) bool {
	articles, err := a.FetchGlobalNews(ctx, 1)
	if err != nil {
		slog.Debug("eastmoney_global_news unavailable", "error", err)
		return false
	}
	return len(articles) > 0
}

// FetchStockNews returns global news filtered by keyword (symbol search).
// Note: This is a keyword search, not symbol-specific filtering. For dedicated
// per-stock news, use EastMoneyNewsAdapter.
func (a *EastMoneyGlobalNewsAdapter) FetchStockNews(ctx context.Context, symbol string, limit int) ([]NewsArticle, error) {
	all, err := a.FetchGlobalNews(ctx, limit*3) // fetch more, filter locally
	if err != nil {
		return nil, err
	}

	// Filter articles mentioning the symbol
	filtered := make([]NewsArticle, 0, limit)
	upperSymbol := strings.ToUpper(symbol)
	for _, art := range all {
		if strings.Contains(strings.ToUpper(art.Title), upperSymbol) ||
			strings.Contains(strings.ToUpper(art.Content), upperSymbol) {
			art.Symbol = symbol
			filtered = append(filtered, art)
			if len(filtered) >= limit {
				break
			}
		}
	}
	return filtered, nil
}

// FetchGlobalNews fetches the latest 7×24 rolling financial news.
func (a *EastMoneyGlobalNewsAdapter) FetchGlobalNews(ctx context.Context, limit int) ([]NewsArticle, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	a.limiter.Wait()

	url := fmt.Sprintf(
		"https://np-weblist.eastmoney.com/comm/web/getFastNewsList"+
			"?client=web&biz=web_724&fastColumn=102&sortEnd="+
			"&pageSize=%d&req_trace=%s",
		limit, uuid.New().String(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_global_news: %w", err)
	}
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Referer", "https://kuaixun.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_global_news: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney_global_news: http %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			FastNewsList []struct {
				Title   string `json:"title"`
				Summary string `json:"summary"`
				ShowTime string `json:"showTime"`
			} `json:"fastNewsList"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eastmoney_global_news: json: %w", err)
	}

	articles := make([]NewsArticle, 0, len(result.Data.FastNewsList))
	for _, item := range result.Data.FastNewsList {
		summary := item.Summary
		if len(summary) > 300 {
			summary = summary[:300]
		}
		articles = append(articles, NewsArticle{
			Symbol:  "",
			Title:   item.Title,
			Content: summary,
			Time:    item.ShowTime,
			Source:  "eastmoney_global",
		})
	}

	return articles, nil
}
