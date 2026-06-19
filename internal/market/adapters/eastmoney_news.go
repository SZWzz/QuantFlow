package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// EastMoneyNewsAdapter fetches stock-specific news from EastMoney's search API.
// Implements NewsAdapter. Based on the a-stock-data SKILL §5.1 (search-api-web.eastmoney.com).
//
// Rate-limited via the shared EastMoney rate limiter to avoid IP bans.
type EastMoneyNewsAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
}

// Compile-time interface check.
var _ NewsAdapter = (*EastMoneyNewsAdapter)(nil)

// NewEastMoneyNewsAdapter creates a new EastMoney news adapter.
func NewEastMoneyNewsAdapter() *EastMoneyNewsAdapter {
	return &EastMoneyNewsAdapter{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneyNewsAdapter) Name() string { return "eastmoney_news" }

func (a *EastMoneyNewsAdapter) IsAvailable(ctx context.Context) bool {
	// Quick check: fetch 1 article for a known symbol and verify non-empty.
	articles, err := a.FetchStockNews(ctx, "600519", 1)
	if err != nil {
		slog.Debug("eastmoney_news unavailable", "error", err)
		return false
	}
	return len(articles) > 0
}

// FetchStockNews fetches news for a single stock by keyword search.
// Uses EastMoney's JSONP search endpoint. Symbol can be any 6-digit CN code.
func (a *EastMoneyNewsAdapter) FetchStockNews(ctx context.Context, symbol string, limit int) ([]NewsArticle, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	a.limiter.Wait()

	// Build JSONP request body (matching SKILL.md §5.1)
	innerParams, _ := json.Marshal(map[string]any{
		"uid":          "",
		"keyword":      symbol,
		"type":         []string{"cmsArticleWebOld"},
		"client":       "web",
		"clientType":   "web",
		"clientVersion": "curr",
		"param": map[string]any{
			"cmsArticleWebOld": map[string]any{
				"searchScope": "default",
				"sort":        "default",
				"pageIndex":   1,
				"pageSize":    limit,
				"preTag":      "",
				"postTag":     "",
			},
		},
	})

	url := fmt.Sprintf(
		"https://search-api-web.eastmoney.com/search/jsonp?cb=jQuery_news&param=%s",
		url.QueryEscape(string(innerParams)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_news: %w", err)
	}
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Referer", "https://so.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_news: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney_news: http %d", resp.StatusCode)
	}

	return a.parseJSONP(resp, symbol, limit)
}

// parseJSONP extracts the JSON payload from EastMoney's JSONP response.
func (a *EastMoneyNewsAdapter) parseJSONP(resp *http.Response, symbol string, limit int) ([]NewsArticle, error) {
	// JSONP format: jQuery_news({...})
	var body strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	text := body.String()

	// Strip JSONP wrapper
	start := strings.Index(text, "(")
	end := strings.LastIndex(text, ")")
	if start < 0 || end < 0 || start >= end {
		return nil, fmt.Errorf("eastmoney_news: invalid JSONP response")
	}
	jsonStr := text[start+1 : end]

	var result struct {
		Result struct {
			CmsArticleWebOld []struct {
				Title     string `json:"title"`
				Content   string `json:"content"`
				Date      string `json:"date"`
				MediaName string `json:"mediaName"`
				URL       string `json:"url"`
			} `json:"cmsArticleWebOld"`
		} `json:"result"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("eastmoney_news: json parse: %w", err)
	}

	stripHTML := regexp.MustCompile(`<[^>]+>`)
	articles := make([]NewsArticle, 0, len(result.Result.CmsArticleWebOld))
	for _, a := range result.Result.CmsArticleWebOld {
		content := stripHTML.ReplaceAllString(a.Content, "")
		if len(content) > 500 {
			content = content[:500]
		}
		articles = append(articles, NewsArticle{
			Symbol:  symbol,
			Title:   stripHTML.ReplaceAllString(a.Title, ""),
			Content: content,
			Time:    a.Date,
			Source:  a.MediaName,
			URL:     a.URL,
		})
	}

	return articles, nil
}

// emUA is the standard User-Agent header required by EastMoney APIs.
const emUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
