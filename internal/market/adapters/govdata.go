// internal/market/adapters/govdata.go
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const fredBaseURL = "https://api.stlouisfed.org/fred"
const secEdgarBaseURL = "https://efts.sec.gov/LATEST/search-index"

// GovDataAdapter defines the interface for government/alternative economic data.
// It covers FRED economic indicators and SEC EDGAR corporate filings.
type GovDataAdapter interface {
	Name() string
	IsAvailable(ctx context.Context) bool

	// FetchIndicator returns observations for a FRED series.
	// seriesID like "GDP", "CPIAUCSL", "UNRATE".
	FetchIndicator(ctx context.Context, seriesID string, limit int) ([]IndicatorPoint, error)

	// FetchCompanyFilings returns recent SEC filings for a CIK code.
	FetchCompanyFilings(ctx context.Context, cik string, limit int) ([]SECFiling, error)
}

// IndicatorPoint represents a single economic indicator observation.
type IndicatorPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// SECFiling represents a single SEC EDGAR filing entry.
type SECFiling struct {
	Company    string `json:"company"`
	FormType   string `json:"form_type"`   // 10-K, 10-Q, 8-K, etc.
	FilingDate string `json:"filing_date"`
	URL        string `json:"url"`
}

// IndicatorMeta holds metadata for a FRED economic indicator.
type IndicatorMeta struct {
	ID       string `json:"id"`       // FRED series ID
	Name     string `json:"name"`     // English name
	NameCN   string `json:"name_cn"`  // Chinese name
	Unit     string `json:"unit"`     // %, USD, Thousands, etc.
	Category string `json:"category"` // gdp, inflation, employment, rates, energy, housing
}

// FREDIndicators contains the 15 predefined economic indicators with metadata.
var FREDIndicators = map[string]IndicatorMeta{
	"GDP": {
		ID: "GDP", Name: "Gross Domestic Product", NameCN: "国内生产总值(GDP)",
		Unit: "Billions of Dollars", Category: "gdp",
	},
	"GDPC1": {
		ID: "GDPC1", Name: "Real Gross Domestic Product", NameCN: "实际GDP",
		Unit: "Billions of Chained 2017 Dollars", Category: "gdp",
	},
	"CPIAUCSL": {
		ID: "CPIAUCSL", Name: "Consumer Price Index for All Urban Consumers", NameCN: "消费者价格指数(CPI)",
		Unit: "Index 1982-1984=100", Category: "inflation",
	},
	"PCEPI": {
		ID: "PCEPI", Name: "Personal Consumption Expenditures Price Index", NameCN: "个人消费支出价格指数(PCE)",
		Unit: "Index 2017=100", Category: "inflation",
	},
	"PPIACO": {
		ID: "PPIACO", Name: "Producer Price Index by Commodity", NameCN: "生产者价格指数(PPI)",
		Unit: "Index 1982=100", Category: "inflation",
	},
	"UNRATE": {
		ID: "UNRATE", Name: "Unemployment Rate", NameCN: "失业率",
		Unit: "%", Category: "employment",
	},
	"PAYEMS": {
		ID: "PAYEMS", Name: "Total Nonfarm Payrolls", NameCN: "非农就业人数",
		Unit: "Thousands", Category: "employment",
	},
	"IC4WSA": {
		ID: "IC4WSA", Name: "Initial Claims", NameCN: "初请失业金人数",
		Unit: "Number", Category: "employment",
	},
	"FEDFUNDS": {
		ID: "FEDFUNDS", Name: "Federal Funds Effective Rate", NameCN: "联邦基金利率",
		Unit: "%", Category: "rates",
	},
	"DGS10": {
		ID: "DGS10", Name: "10-Year Treasury Constant Maturity Rate", NameCN: "10年期国债收益率",
		Unit: "%", Category: "rates",
	},
	"T10Y2Y": {
		ID: "T10Y2Y", Name: "10-Year Treasury Minus 2-Year Treasury", NameCN: "美债10Y-2Y利差",
		Unit: "%", Category: "rates",
	},
	// Real-time commodity prices (oil, gas) now served by App.GetCommodityQuotes via Sina futures.
	"HOUST": {
		ID: "HOUST", Name: "Housing Starts: Total", NameCN: "新屋开工",
		Unit: "Thousands of Units", Category: "housing",
	},
	"MSPUS": {
		ID: "MSPUS", Name: "Median Sales Price of Houses Sold", NameCN: "房屋销售中位价",
		Unit: "Dollars", Category: "housing",
	},
}

// GovDataHTTPAdapter fetches economic indicator data from the FRED API
// and SEC filings from EDGAR. It reads the FRED_API_KEY env var; without
// it IsAvailable() returns false and all methods fall back gracefully.
type GovDataHTTPAdapter struct {
	client     *http.Client
	apiKey     string
	indicators map[string]IndicatorMeta
}

// Compile-time interface check.
var _ GovDataAdapter = (*GovDataHTTPAdapter)(nil)

// NewGovDataAdapter creates a new GovData HTTP adapter.
// Call SetAPIKey() later to configure the FRED API key from app config.
func NewGovDataAdapter() *GovDataHTTPAdapter {
	return &GovDataHTTPAdapter{
		client:     &http.Client{Timeout: 30 * time.Second},
		apiKey:     os.Getenv("FRED_API_KEY"), // env fallback
		indicators: FREDIndicators,
	}
}

// SetAPIKey updates the FRED API key (called from app config after startup).
func (a *GovDataHTTPAdapter) SetAPIKey(key string) {
	if key != "" {
		a.apiKey = key
	}
}

func (a *GovDataHTTPAdapter) Name() string { return "govdata" }

// IsAvailable returns true only when FRED_API_KEY is set and the FRED API is reachable.
func (a *GovDataHTTPAdapter) IsAvailable(ctx context.Context) bool {
	if a.apiKey == "" {
		slog.Debug("govdata: FRED_API_KEY not set, govdata adapter unavailable")
		return false
	}
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/series/observations?series_id=GDP&api_key=%s&file_type=json&limit=1", fredBaseURL, a.apiKey), nil)
	resp, err := a.client.Do(req)
	if err != nil {
		slog.Debug("govdata: FRED availability check failed", "error", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// FetchIndicator returns observations for a FRED series.
// seriesID like "GDP", "CPIAUCSL", "UNRATE".
// Uses the last 12 months as the default observation range.
func (a *GovDataHTTPAdapter) FetchIndicator(ctx context.Context, seriesID string, limit int) ([]IndicatorPoint, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("govdata: FRED_API_KEY not set")
	}
	if limit <= 0 {
		limit = 12
	}
	if limit > 240 {
		limit = 240
	}

	// Default to last 3 years to get enough data points
	now := time.Now()
	startDate := now.AddDate(-3, 0, 0).Format("2006-01-02")

	url := fmt.Sprintf("%s/series/observations?series_id=%s&api_key=%s&file_type=json&sort_order=desc&limit=%d&observation_start=%s",
		fredBaseURL, seriesID, a.apiKey, limit, startDate)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("govdata: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("govdata: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("govdata: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// FRED API returns: {"observations": [{"date": "2026-06-15", "value": "2.5"}, ...]}
	var raw struct {
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("govdata: parse error: %w", err)
	}

	points := make([]IndicatorPoint, 0, len(raw.Observations))
	for _, obs := range raw.Observations {
		// Handle "." values (missing data) by skipping
		if obs.Value == "." || strings.TrimSpace(obs.Value) == "" {
			continue
		}
		val, err := strconv.ParseFloat(obs.Value, 64)
		if err != nil {
			slog.Debug("govdata: skipping unparseable value", "series", seriesID, "date", obs.Date, "value", obs.Value, "error", err)
			continue
		}
		points = append(points, IndicatorPoint{
			Date:  obs.Date,
			Value: val,
		})
	}

	// Reverse so that dates are oldest-first for time series display.
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	return points, nil
}

// FetchCompanyFilings returns recent SEC filings for a CIK code.
// CIK should be a string like "0000320193" (Apple).
// Uses SEC EDGAR public API (no auth required).
func (a *GovDataHTTPAdapter) FetchCompanyFilings(ctx context.Context, cik string, limit int) ([]SECFiling, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	forms := `["10-K","10-Q","8-K"]`
	url := fmt.Sprintf("%s?q=%s&forms=%s&limit=%d", secEdgarBaseURL, cik, forms, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("govdata SEC: %w", err)
	}
	req.Header.Set("User-Agent", "QuantFlow/1.0 (contact@example.com)")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("govdata SEC: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("govdata SEC: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// SEC EDGAR response shape: nested "hits" → "hits" array
	var raw struct {
		Hits struct {
			Hits []struct {
				Source struct {
					CompanyName    string `json:"company"`
					FormType       string `json:"form"`
					FilingDate     string `json:"date"`
					URL            string `json:"url"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("govdata SEC: parse error: %w", err)
	}

	filings := make([]SECFiling, 0, len(raw.Hits.Hits))
	for _, hit := range raw.Hits.Hits {
		s := hit.Source
		filings = append(filings, SECFiling{
			Company:    s.CompanyName,
			FormType:   s.FormType,
			FilingDate: s.FilingDate,
			URL:        s.URL,
		})
	}
	return filings, nil
}
