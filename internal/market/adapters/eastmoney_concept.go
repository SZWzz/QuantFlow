package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ConceptBlock represents a concept/industry/region block a stock belongs to.
// EastMoney mixes industry, concept, and region blocks in one list; the Name
// field is self-explanatory (e.g. "白酒"=industry, "贵州板块"=region, "酿酒概念"=concept).
type ConceptBlock struct {
	Name      string  `json:"name"`       // 板块名称
	Code      string  `json:"code"`       // BK板块代码 e.g. "BK0477"
	ChangePct float64 `json:"change_pct"` // 板块当日涨跌幅%
	LeadStock string  `json:"lead_stock"` // 板块龙头股简称
}

// EastMoneyConceptAdapter fetches stock concept/industry/region affiliations.
// Based on a-stock-data SKILL §3.3 (EastMoney push2 slist, spt=3).
//
// One request returns ALL blocks a stock belongs to (up to ~30 for major stocks).
type EastMoneyConceptAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
}

// NewEastMoneyConceptAdapter creates a new concept blocks adapter.
func NewEastMoneyConceptAdapter() *EastMoneyConceptAdapter {
	return &EastMoneyConceptAdapter{
		client:  &http.Client{Timeout: 15 * time.Second},
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneyConceptAdapter) Name() string { return "eastmoney_concept" }

func (a *EastMoneyConceptAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchConceptBlocks(ctx, "600519")
	return err == nil
}

// FetchConceptBlocks returns all concept/industry/region blocks for a stock.
// code: 6-digit CN stock code.
func (a *EastMoneyConceptAdapter) FetchConceptBlocks(ctx context.Context, code string) ([]ConceptBlock, error) {
	a.limiter.Wait()

	marketCode := "1" // Shanghai
	if !(code[0] == '6' || code[0] == '9') {
		marketCode = "0" // Shenzhen/Beijing
	}

	url := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/slist/get"+
			"?fltt=2&invt=2&secid=%s.%s&spt=3&pi=0&pz=200&po=1"+
			"&fields=f12,f14,f3,f128",
		marketCode, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_concept: %w", err)
	}
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_concept: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney_concept: http %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Diff map[string]struct {
				F12  string      `json:"f12"`  // BK code
				F14  string      `json:"f14"`  // name
				F3   interface{} `json:"f3"`   // change%
				F128 string      `json:"f128"` // lead stock
			} `json:"diff"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eastmoney_concept: json: %w", err)
	}

	blocks := make([]ConceptBlock, 0, len(result.Data.Diff))
	for _, d := range result.Data.Diff {
		blocks = append(blocks, ConceptBlock{
			Name:      d.F14,
			Code:      d.F12,
			ChangePct: toFloat(d.F3),
			LeadStock: d.F128,
		})
	}

	slog.Debug("eastmoney_concept fetched", "code", code, "blocks", len(blocks))
	return blocks, nil
}
