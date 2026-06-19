package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"quantflow/internal/market"
)

// SinaAdapter fetches A/HK market data from Sina Finance (free, no auth).
type SinaAdapter struct {
	client *http.Client
}

func NewSinaAdapter() *SinaAdapter {
	return &SinaAdapter{client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *SinaAdapter) Name() string      { return "sina" }
func (a *SinaAdapter) Markets() []string  { return []string{"CN", "HK"} }
func (a *SinaAdapter) RequiresAuth() bool { return false }

func (a *SinaAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://hq.sinajs.cn/list=sh600519", nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *SinaAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	code := toSinaCode(symbol)
	url := fmt.Sprintf("http://hq.sinajs.cn/list=%s", code)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("sina: http error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sina: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return parseSinaQuote(symbol, string(body))
}

func (a *SinaAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, start, end int64) ([]market.OHLCVBar, error) {
	return nil, fmt.Errorf("sina: OHLCV not supported (real-time quotes only, use tushare or yahoo for historical data)")
}

func (a *SinaAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}
