package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"quantflow/internal/market"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// MootdxAdapter fetches A-share data via TDX (通达信) TCP protocol using the
// Python mootdx library. This is the primary free A-share data source — no
// registration, no API key, no IP-blocking risk.
//
// Architecture: Go adapter → gRPC → Python sidecar → mootdx → TDX TCP server.
type MootdxAdapter struct {
	dataClient *python.DataClient
}

// NewMootdxAdapter creates a new Mootdx adapter.
// Pass nil dataClient if the Python sidecar is unavailable;
// the adapter will report IsAvailable() == false and return clear errors.
func NewMootdxAdapter(dataClient *python.DataClient) *MootdxAdapter {
	return &MootdxAdapter{dataClient: dataClient}
}

func (a *MootdxAdapter) Name() string      { return "mootdx" }
func (a *MootdxAdapter) Markets() []string  { return []string{"CN"} }
func (a *MootdxAdapter) RequiresAuth() bool { return false }

func (a *MootdxAdapter) IsAvailable(ctx context.Context) bool {
	// Cheap check only: a live TDX round-trip here would double the connections
	// per quote (the registry probes IsAvailable, then calls FetchQuote). The real
	// liveness signal is FetchQuote itself; on failure the fallback chain moves to
	// the next adapter. So we report available whenever a DataClient is wired.
	return a.dataClient != nil
}

// ── Quote ──────────────────────────────────────────────────────────────────────

func (a *MootdxAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("mootdx: Python sidecar not connected")
	}

	resp, err := a.dataClient.FetchData(ctx, &pb.FetchDataRequest{
		Source:   "mootdx",
		DataType: "quote",
		Symbols:  []string{symbol},
	})
	if err != nil {
		return nil, err
	}

	var quotes []mootdxQuote
	if err := json.Unmarshal(resp.Data, &quotes); err != nil {
		return nil, fmt.Errorf("mootdx: parse quote response: %w", err)
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("mootdx: no quote data for %s", symbol)
	}

	q := quotes[0]
	change := q.Last - q.Open
	changePct := 0.0
	if q.Open > 0 {
		changePct = (change / q.Open) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    q.Symbol,
		Last:      q.Last,
		Open:      q.Open,
		High:      q.High,
		Low:       q.Low,
		Volume:    q.Volume,
		Change:    change,
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// ── OHLCV ──────────────────────────────────────────────────────────────────────

func (a *MootdxAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("mootdx: Python sidecar not connected")
	}

	params := map[string]string{"interval": interval}
	if fqfactor != "" {
		if fqfactor != "qfq" && fqfactor != "hfq" {
			return nil, fmt.Errorf("mootdx: invalid fqfactor %q, must be one of: \"\" (不复权), \"qfq\" (前复权), \"hfq\" (后复权)", fqfactor)
		}
		params["fqfactor"] = fqfactor
	}

	resp, err := a.dataClient.FetchData(ctx, &pb.FetchDataRequest{
		Source:    "mootdx",
		DataType:  "ohlcv",
		Symbols:   []string{symbol},
		StartDate: time.Unix(start, 0).Format("2006-01-02"),
		EndDate:   time.Unix(end, 0).Format("2006-01-02"),
		Params:    params,
	})
	if err != nil {
		return nil, err
	}

	var rawBars []mootdxOHLCVBar
	if err := json.Unmarshal(resp.Data, &rawBars); err != nil {
		return nil, fmt.Errorf("mootdx: parse ohlcv response: %w", err)
	}

	bars := make([]market.OHLCVBar, 0, len(rawBars))
	for _, b := range rawBars {
		bars = append(bars, market.OHLCVBar{
			Symbol: b.Symbol,
			Date:   b.Date,
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: b.Volume,
		})
	}

	return bars, nil
}

func (a *MootdxAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// ── F10 categories ─────────────────────────────────────────────────────────

// F10Categories returns the 9 supported F10 data categories.
func F10Categories() []string {
	return []string{"最新提示", "公司概况", "财务分析", "股东研究", "股本结构", "资本运作", "业内点评", "行业分析", "公司大事"}
}

// ── Finance ───────────────────────────────────────────────────────────────────

// MootdxFinance holds the 37-field quarterly finance snapshot from TDX.
// Key fields are parsed strongly; remaining fields are kept in Raw.
type MootdxFinance struct {
	Symbol string  `json:"symbol"`
	EPS    float64 `json:"eps"`    // 每股收益
	BVPS   float64 `json:"bvps"`   // 每股净资产
	ROE    float64 `json:"roe"`    // 净资产收益率(%)
	Profit float64 `json:"profit"` // 净利润(元)
	Income float64 `json:"income"` // 主营收入(元)
	// Raw holds all 37+ fields from mootdx verbatim.
	Raw map[string]string `json:"raw"`
}

// FetchFinance fetches quarterly finance snapshot data via mootdx (37 fields).
func (a *MootdxAdapter) FetchFinance(ctx context.Context, symbol string) (*MootdxFinance, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("mootdx: Python sidecar not connected")
	}

	resp, err := a.dataClient.FetchData(ctx, &pb.FetchDataRequest{
		Source:   "mootdx",
		DataType: "finance",
		Symbols:  []string{symbol},
	})
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &results); err != nil {
		return nil, fmt.Errorf("mootdx finance: parse: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("mootdx: no finance data for %s", symbol)
	}

	r := results[0]
	fin := &MootdxFinance{
		Symbol: symbol,
		Raw:    make(map[string]string),
	}
	for k, v := range r {
		s := fmt.Sprint(v)
		fin.Raw[k] = s
		switch k {
		// Chinese keys (from old Python sidebar)
		case "每股收益":
			fin.EPS = parseFloatSafe(s)
		case "每股净资产":
			fin.BVPS = parseFloatSafe(s)
		case "净资产收益率":
			fin.ROE = parseFloatSafe(s)
		case "净利润":
			fin.Profit = parseFloatSafe(s)
		case "主营收入":
			fin.Income = parseFloatSafe(s)
		// Pinyin keys (mootdx 0.11.x direct output)
		case "jinglirun":
			fin.Profit = parseFloatSafe(s)
		case "zhuyingshouru":
			fin.Income = parseFloatSafe(s)
		case "meigujingzichan":
			fin.BVPS = parseFloatSafe(s)
		case "meigushouyi":
			fin.EPS = parseFloatSafe(s)
		}
	}
	// Derive EPS if not directly provided (works for both positive and negative profit)
	if fin.EPS == 0 && fin.Profit != 0 {
		if zgb, ok := r["zongguben"]; ok {
			total := toFloatMootdx(zgb)
			if total > 0 {
				fin.EPS = fin.Profit / total
			}
		}
	}
	if fin.ROE == 0 && fin.Profit > 0 {
		if jzc, ok := r["jingzichan"]; ok {
			equity := toFloatMootdx(jzc)
			if equity > 0 {
				fin.ROE = fin.Profit / equity * 100
			}
		}
	}
	return fin, nil
}

func toFloatMootdx(v interface{}) float64 {
	return parseFloatSafe(fmt.Sprint(v))
}

// ── F10 ──────────────────────────────────────────────────────────────────────

// FetchF10 fetches company text data from TDX F10.
// category must be one of the 9 F10Categories().
func (a *MootdxAdapter) FetchF10(ctx context.Context, symbol, category string) (string, error) {
	if a.dataClient == nil {
		return "", fmt.Errorf("mootdx: Python sidecar not connected")
	}

	resp, err := a.dataClient.FetchData(ctx, &pb.FetchDataRequest{
		Source:   "mootdx",
		DataType: "f10",
		Symbols:  []string{symbol},
		Params:   map[string]string{"category": category},
	})
	if err != nil {
		return "", err
	}

	var results []struct {
		Symbol   string `json:"symbol"`
		Category string `json:"category"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(resp.Data, &results); err != nil {
		return "", fmt.Errorf("mootdx f10: parse: %w", err)
	}
	if len(results) == 0 || results[0].Text == "" {
		return "", fmt.Errorf("mootdx: no F10(%s) data for %s", category, symbol)
	}
	return results[0].Text, nil
}

// ── Response types ─────────────────────────────────────────────────────────────

type mootdxQuote struct {
	Symbol string  `json:"symbol"`
	Last   float64 `json:"last"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume float64 `json:"volume"`
}

type mootdxOHLCVBar struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}
