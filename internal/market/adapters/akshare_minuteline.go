package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"quantflow/internal/market"
	"quantflow/internal/python"
	"sync"
	"time"

	pb "quantflow/internal/python/proto"
)

const akshareMinuteCooldown = 3 * time.Second

type AKShareMinuteAdapter struct {
	dataClient *python.DataClient
	mu         sync.Mutex
	lastFetch  map[string]time.Time
}

func NewAKShareMinuteAdapter(dataClient *python.DataClient) *AKShareMinuteAdapter {
	return &AKShareMinuteAdapter{
		dataClient: dataClient,
		lastFetch:  make(map[string]time.Time),
	}
}

func (a *AKShareMinuteAdapter) Name() string       { return "akshare_hk_minute" }
func (a *AKShareMinuteAdapter) Markets() []string  { return []string{"HK"} }
func (a *AKShareMinuteAdapter) RequiresAuth() bool { return false }

func (a *AKShareMinuteAdapter) IsAvailable(ctx context.Context) bool {
	return a.dataClient != nil
}

func (a *AKShareMinuteAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	return nil, fmt.Errorf("akshare_hk_minute: FetchQuote not implemented")
}

func (a *AKShareMinuteAdapter) FetchOHLCV(ctx context.Context, symbol, interval, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	return nil, fmt.Errorf("akshare_hk_minute: FetchOHLCV not implemented")
}

func (a *AKShareMinuteAdapter) HealthCheck(ctx context.Context) error {
	if a.dataClient == nil {
		return fmt.Errorf("akshare_hk_minute: Python sidecar not connected")
	}
	return nil
}

func (a *AKShareMinuteAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("akshare_hk_minute: Python sidecar not connected")
	}

	a.mu.Lock()
	if last, ok := a.lastFetch[symbol]; ok && time.Since(last) < akshareMinuteCooldown {
		a.mu.Unlock()
		return nil, nil
	}
	a.lastFetch[symbol] = time.Now()
	a.mu.Unlock()

	resp, err := a.dataClient.FetchData(context.Background(), &pb.FetchDataRequest{
		Source:   "akshare",
		DataType: "hk_minute",
		Symbols:  []string{symbol},
	})
	if err != nil {
		return nil, fmt.Errorf("akshare_hk_minute: %w", err)
	}

	var raw []struct {
		Time   string  `json:"time"`
		Date   string  `json:"date,omitempty"`
		Close  float64 `json:"close"`
		Volume float64 `json:"volume"`
		Amount float64 `json:"amount,omitempty"`
	}
	if err := json.Unmarshal(resp.Data, &raw); err != nil {
		return nil, fmt.Errorf("akshare_hk_minute parse: %w", err)
	}

	ticks := make([]market.MinuteTick, 0, len(raw))
	for _, r := range raw {
		t := r.Time
		if len(r.Date) >= 16 {
			t = r.Date[11:16]
		} else if len(r.Time) >= 16 {
			t = r.Time[11:16]
		}
		ticks = append(ticks, market.MinuteTick{
			Time:   t,
			Price:  r.Close,
			Volume: r.Volume,
			Amount: r.Amount,
		})
	}
	return ticks, nil
}
