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

func (a *MootdxAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("mootdx: Python sidecar not connected")
	}

	resp, err := a.dataClient.FetchData(ctx, &pb.FetchDataRequest{
		Source:    "mootdx",
		DataType:  "ohlcv",
		Symbols:   []string{symbol},
		StartDate: time.Unix(start, 0).Format("2006-01-02"),
		EndDate:   time.Unix(end, 0).Format("2006-01-02"),
		Params:    map[string]string{"interval": interval},
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
