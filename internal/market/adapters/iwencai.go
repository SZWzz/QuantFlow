package adapters

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// IwencaiArticle represents a single search result from iwencai semantic search.
type IwencaiArticle struct {
	UID         string      `json:"uid"`
	Title       string      `json:"title"`
	PublishDate string      `json:"publish_date"`
	Channel     string      `json:"channel"` // "report" / "announcement" / "news"
	ScoreRaw    interface{} `json:"score"`   // number or string-encoded float
	Content     string      `json:"content"` // snippet or full text
	OrgName     string      `json:"org_name"`     // 机构名称 (from extra.organization)
	PDFURL      string      `json:"pdf_url"`      // PDF link (from extra.pdf_url)
	Author      string      `json:"author"`       // 作者 (from extra.author)
}

// Score returns the relevance score as a string for comparison.
func (a IwencaiArticle) Score() string {
	switch v := a.ScoreRaw.(type) {
	case float64:
		return fmt.Sprintf("%.4f", v)
	case string:
		return v
	default:
		return "0"
	}
}

// IwencaiQueryRow represents one row from an iwencai structured query.
type IwencaiQueryRow struct {
	Fields map[string]string `json:"fields"` // column name → value
}

// IwencaiAdapter provides NL semantic search over research reports,
// announcements, and news via the iwencai (爱问财) OpenAPI.
//
// Requires IWENCAI_API_KEY environment variable.
// Optional IWENCAI_BASE_URL (defaults to https://openapi.iwencai.com).
//
// Based on a-stock-data SKILL §2.3.
type IwencaiAdapter struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// NewIwencaiAdapter creates a new iwencai adapter.
// Reads IWENCAI_API_KEY and IWENCAI_BASE_URL from environment.
func NewIwencaiAdapter() *IwencaiAdapter {
	baseURL := os.Getenv("IWENCAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openapi.iwencai.com"
	}
	return &IwencaiAdapter{
		client:  &http.Client{Timeout: 30 * time.Second},
		apiKey:  os.Getenv("IWENCAI_API_KEY"),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Name returns the adapter identifier.
func (a *IwencaiAdapter) Name() string { return "iwencai" }

// IsAvailable checks whether the API key is configured and the endpoint is reachable.
func (a *IwencaiAdapter) IsAvailable(ctx context.Context) bool {
	if a.apiKey == "" {
		return false
	}
	// Quick check: search for a known stock, just 1 result
	results, err := a.Search(ctx, "贵州茅台", "report", 1)
	if err != nil {
		slog.Debug("iwencai unavailable", "error", err)
		return false
	}
	return len(results) > 0
}

// RequiresAuth returns true — iwencai always needs an API key.
func (a *IwencaiAdapter) RequiresAuth() bool { return true }

// Search performs a semantic search across the specified channel.
// channel: "report" (研报), "announcement" (公告), "news" (新闻).
// size: max results (capped at 50 by upstream).
// Results are deduplicated by UID (highest score kept).
func (a *IwencaiAdapter) Search(ctx context.Context, query string, channel string, size int) ([]IwencaiArticle, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("iwencai: IWENCAI_API_KEY not set — get a free key at https://www.iwencai.com/skillhub")
	}
	if size <= 0 {
		size = 10
	}
	if size > 50 {
		size = 50
	}

	payload := map[string]interface{}{
		"channels": []string{channel},
		"app_id":   "AIME_SKILL",
		"query":    query,
		"size":     size,
	}

	body, err := a.post(ctx, "/v1/comprehensive/search", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		StatusCode int              `json:"status_code"`
		StatusMsg  string           `json:"status_msg"`
		Data       []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("iwencai search: json: %w", err)
	}
	if result.StatusCode != 0 {
		return nil, fmt.Errorf("iwencai search: %s (code %d)", result.StatusMsg, result.StatusCode)
	}

	articles := make([]IwencaiArticle, 0, len(result.Data))
	for _, raw := range result.Data {
		// Try direct unmarshal first
		var art IwencaiArticle
		if err := json.Unmarshal(raw, &art); err != nil {
			slog.Warn("iwencai: failed to parse article", "error", err)
			continue
		}
		// Parse extra field (may be string or object)
		var wrapper struct {
			Extra json.RawMessage `json:"extra"`
		}
		if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Extra) > 0 {
			a.parseExtra(&art, wrapper.Extra)
		}
		articles = append(articles, art)
	}

	return dedupArticles(articles), nil
}

// Query runs a natural-language structured data query.
// Example: "贵州茅台 ROE" returns DataFrame-like rows with fields.
func (a *IwencaiAdapter) Query(ctx context.Context, query string, page, limit int) ([]IwencaiQueryRow, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("iwencai: IWENCAI_API_KEY not set — get a free key at https://www.iwencai.com/skillhub")
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	payload := map[string]interface{}{
		"query":        query,
		"page":         fmt.Sprintf("%d", page),
		"limit":        fmt.Sprintf("%d", limit),
		"is_cache":     "1",
		"expand_index": "true",
	}

	body, err := a.post(ctx, "/v1/query2data", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		StatusCode int              `json:"status_code"`
		StatusMsg  string           `json:"status_msg"`
		Datas      []json.RawMessage `json:"datas"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("iwencai query: json: %w", err)
	}
	if result.StatusCode != 0 {
		return nil, fmt.Errorf("iwencai query: %s (code %d)", result.StatusMsg, result.StatusCode)
	}

	rows := make([]IwencaiQueryRow, 0, len(result.Datas))
	for _, raw := range result.Datas {
		var fields map[string]string
		if err := json.Unmarshal(raw, &fields); err != nil {
			slog.Warn("iwencai: failed to parse query row", "error", err)
			continue
		}
		rows = append(rows, IwencaiQueryRow{Fields: fields})
	}

	return rows, nil
}

// ── Helpers ────────────────────────────────────────────────────────────

// post sends a JSON POST request to the iwencai API with required headers.
func (a *IwencaiAdapter) post(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("iwencai: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("iwencai: request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// SkillHub 2.0 required X-Claw headers
	for k, v := range clawHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iwencai: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var msg [200]byte
		n, _ := resp.Body.Read(msg[:])
		return nil, fmt.Errorf("iwencai HTTP %d: %s", resp.StatusCode, string(msg[:n]))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("iwencai: read body: %w", err)
	}
	return raw, nil
}

// clawHeaders returns the SkillHub 2.0 X-Claw authentication headers.
func clawHeaders() map[string]string {
	traceID := make([]byte, 32)
	rand.Read(traceID)
	return map[string]string{
		"X-Claw-Call-Type":      "normal",
		"X-Claw-Skill-Id":       "report-search",
		"X-Claw-Skill-Version":  "2.0.0",
		"X-Claw-Plugin-Id":      "none",
		"X-Claw-Plugin-Version": "none",
		"X-Claw-Trace-Id":       hex.EncodeToString(traceID),
	}
}

// parseExtra fills OrgName, PDFURL, and Author from the extra field.
// The extra field can be either a JSON object or a JSON-encoded string.
func (a *IwencaiAdapter) parseExtra(art *IwencaiArticle, raw json.RawMessage) {
	var extra map[string]string

	// Try as object first
	if err := json.Unmarshal(raw, &extra); err != nil {
		// Try as string (double-encoded)
		var s string
		if err2 := json.Unmarshal(raw, &s); err2 == nil {
			_ = json.Unmarshal([]byte(s), &extra)
		}
	}

	if extra == nil {
		return
	}
	if v, ok := extra["organization"]; ok && art.OrgName == "" {
		art.OrgName = v
	}
	if v, ok := extra["pdf_url"]; ok && art.PDFURL == "" {
		art.PDFURL = v
	}
	if v, ok := extra["author"]; ok && art.Author == "" {
		art.Author = v
	}
}

// dedupArticles removes duplicate articles (same UID), keeping the one with
// the highest score. Results are sorted by publish_date descending.
func dedupArticles(articles []IwencaiArticle) []IwencaiArticle {
	best := make(map[string]IwencaiArticle)
	for _, a := range articles {
		uid := a.UID
		if uid == "" {
			uid = fmt.Sprintf("%s|%s", a.Title, a.PublishDate)
		}
		if existing, ok := best[uid]; !ok || scoreVal(a.Score()) > scoreVal(existing.Score()) {
			best[uid] = a
		}
	}

	result := make([]IwencaiArticle, 0, len(best))
	for _, v := range best {
		result = append(result, v)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].PublishDate > result[j].PublishDate
	})

	return result
}

// scoreVal parses a score string to float64 for comparison.
func scoreVal(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
