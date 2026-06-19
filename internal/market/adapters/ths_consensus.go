package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ConsensusEPS holds analyst consensus EPS forecast data.
type ConsensusEPS struct {
	Year         string  `json:"year"`         // 预测年度
	AnalystCount int     `json:"analyst_count"` // 预测机构数
	MinEPS       float64 `json:"min_eps"`       // 最小值
	AvgEPS       float64 `json:"avg_eps"`       // 均值(一致预期)
	MaxEPS       float64 `json:"max_eps"`       // 最大值
}

// THSConsensusAdapter fetches analyst consensus EPS forecasts from 同花顺.
// Based on a-stock-data SKILL §2.2 (basic.10jqka.com.cn, HTML table parsing).
//
// Important: only stocks with ≥3 analyst coverage have meaningful data.
// Small-cap and newly-listed stocks typically return empty.
type THSConsensusAdapter struct {
	client *http.Client
}

// NewTHSConsensusAdapter creates a new consensus EPS adapter.
func NewTHSConsensusAdapter() *THSConsensusAdapter {
	return &THSConsensusAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *THSConsensusAdapter) Name() string { return "ths_consensus" }

func (a *THSConsensusAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchConsensus(ctx, "600519")
	return err == nil
}

// FetchConsensus fetches analyst consensus EPS for a stock.
// Returns empty slice if the stock has no analyst coverage.
func (a *THSConsensusAdapter) FetchConsensus(ctx context.Context, code string) ([]ConsensusEPS, error) {
	url := fmt.Sprintf("https://basic.10jqka.com.cn/new/%s/worth.html", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ths_consensus: %w", err)
	}
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://basic.10jqka.com.cn/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ths_consensus: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ths_consensus: http %d", resp.StatusCode)
	}

	// THS page is GBK-encoded — decode before HTML parsing.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ths_consensus: read: %w", err)
	}
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Body, _, err := transform.Bytes(decoder, body)
	if err != nil {
		// Fallback: try as UTF-8 (some newer THS pages may have switched)
		slog.Debug("ths_consensus: GBK decode failed, trying UTF-8", "error", err)
		utf8Body = body
	}

	doc, err := html.Parse(bytes.NewReader(utf8Body))
	if err != nil {
		return nil, fmt.Errorf("ths_consensus: parse: %w", err)
	}

	// Find tables containing consensus EPS data.
	// THS uses labels like "每股收益", "一致预期", "均值", "预测".
	tables := extractRows(doc)
	for _, rows := range tables {
		tableText := strings.Join(rows, "\n")
		if strings.Contains(tableText, "每股收益") ||
			strings.Contains(tableText, "一致预期") ||
			strings.Contains(tableText, "预测机构") ||
			strings.Contains(tableText, "均值") {
			result := parseConsensusRows(rows)
			if len(result) > 0 {
				slog.Debug("ths_consensus: found consensus table", "code", code, "years", len(result))
				return result, nil
			}
		}
	}

	slog.Debug("ths_consensus: no consensus table found (likely no analyst coverage)",
		"code", code)
	return nil, nil
}

// ── HTML table extraction (row-based, GBK-safe) ───────────────────────

// extractRows finds every HTML table and returns its rows as joined-cell text.
func extractRows(n *html.Node) [][]string {
	var tables [][]string
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "table" {
			tables = append(tables, extractTableRows(node))
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return tables
}

// extractTableRows extracts each tr from a table, joining td/th text.
func extractTableRows(table *html.Node) []string {
	var rows []string
	for tr := table.FirstChild; tr != nil; tr = tr.NextSibling {
		if tr.Type == html.ElementNode && (tr.Data == "tr" || tr.Data == "TR") {
			rows = append(rows, rowText(tr))
		}
	}
	return rows
}

// rowText joins all text nodes within a row, separated by space.
func rowText(tr *html.Node) string {
	var buf strings.Builder
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				if buf.Len() > 0 {
					buf.WriteByte(' ')
				}
				buf.WriteString(text)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(tr)
	return buf.String()
}

// parseConsensusRows parses row texts into ConsensusEPS entries.
// Looks for years (4 digits) followed by numeric fields.
// Common formats:
//
//	年度 预测机构数 最小值 均值 最大值  ...
//	2025 15          3.20   3.80  4.10   ...
func parseConsensusRows(rows []string) []ConsensusEPS {
	var result []ConsensusEPS

	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}

		fields := strings.Fields(row)
		if len(fields) < 2 {
			continue
		}

		// Try each position as a potential year
		for i := 0; i < len(fields)-1; i++ {
			if len(fields[i]) == 4 {
				if year, err := strconv.Atoi(fields[i]); err == nil && year >= 2020 && year <= 2100 {
					eps := ConsensusEPS{Year: fields[i]}
					nums := extractNumbers(fields[i+1:])
					switch {
					case len(nums) >= 4:
						eps.AnalystCount = int(nums[0])
						eps.MinEPS = nums[1]
						eps.AvgEPS = nums[2]
						eps.MaxEPS = nums[3]
					case len(nums) == 3:
						eps.MinEPS = nums[0]
						eps.AvgEPS = nums[1]
						eps.MaxEPS = nums[2]
					case len(nums) == 2:
						eps.AnalystCount = int(nums[0])
						eps.AvgEPS = nums[1]
					case len(nums) == 1:
						eps.AvgEPS = nums[0]
					}
					if eps.AvgEPS > 0 {
						result = append(result, eps)
					}
					break // found year in this row
				}
			}
		}
	}
	return result
}

func extractNumbers(fields []string) []float64 {
	var nums []float64
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if n, err := strconv.ParseFloat(f, 64); err == nil {
			nums = append(nums, n)
			if len(nums) >= 4 {
				break
			}
		}
	}
	return nums
}
