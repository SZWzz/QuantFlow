package adapters

import (
	"context"
	"testing"
)

// TestFetchOHLCV_NoFreeOHLCV_ReturnsError verifies that adapters backed by
// real-time-only APIs return an honest error instead of silently fabricating
// a single fake bar.
func TestFetchOHLCV_NoFreeOHLCV_ReturnsError(t *testing.T) {
	ctx := context.Background()

	t.Run("sina", func(t *testing.T) {
		_, err := NewSinaAdapter().FetchOHLCV(ctx, "600519", "1d", 0, 0)
		if err == nil {
			t.Error("sina: FetchOHLCV should return an error (real-time quotes only, no free OHLCV API)")
		}
	})

	t.Run("akshare", func(t *testing.T) {
		_, err := NewAKShareAdapter().FetchOHLCV(ctx, "600519", "1d", 0, 0)
		if err == nil {
			t.Error("akshare: FetchOHLCV should return an error (Tencent quote API only, no direct OHLCV)")
		}
	})
}

// TestFetchOHLCV_ReturnsErrorWhenNotSupported verifies adapters that are
// honest about not supporting OHLCV continue to do so.
func TestFetchOHLCV_ReturnsErrorWhenNotSupported(t *testing.T) {
	ctx := context.Background()

	t.Run("coingecko", func(t *testing.T) {
		_, err := NewCoinGeckoAdapter().FetchOHLCV(ctx, "BTCUSDT", "1d", 0, 0)
		if err == nil {
			t.Error("coingecko: FetchOHLCV should return an error (not supported on free tier)")
		}
	})

	t.Run("polygon", func(t *testing.T) {
		_, err := NewPolygonAdapter().FetchOHLCV(ctx, "AAPL", "1d", 0, 0)
		if err == nil {
			t.Error("polygon: FetchOHLCV should return an error (not implemented without API key)")
		}
	})
}

// TestFetchOHLCV_KlineAdapters_DoNotPanic verifies that adapters with K-line
// support do NOT return fake "not supported" errors (network errors are OK).
func TestFetchOHLCV_KlineAdapters_DoNotPanic(t *testing.T) {
	ctx := context.Background()

	t.Run("tencent", func(t *testing.T) {
		_, err := NewTencentAdapter().FetchOHLCV(ctx, "600519", "1D", 0, 0)
		if err != nil {
			msg := err.Error()
			if contains(msg, "not supported") || contains(msg, "real-time quotes only") {
				t.Errorf("tencent: should have K-line support, got error: %v", err)
			}
		}
	})

	t.Run("baidu", func(t *testing.T) {
		_, err := NewBaiduAdapter().FetchOHLCV(ctx, "600519", "1D", 0, 0)
		if err != nil {
			msg := err.Error()
			if contains(msg, "not supported") || contains(msg, "quote API only") {
				t.Errorf("baidu: should have K-line support, got error: %v", err)
			}
		}
	})

	t.Run("mootdx", func(t *testing.T) {
		// mootdx requires Python sidecar — without bridge, it returns error
		_, err := NewMootdxAdapter(nil).FetchOHLCV(ctx, "600519", "1D", 0, 0)
		if err != nil {
			msg := err.Error()
			if contains(msg, "not supported") {
				t.Errorf("mootdx: should have K-line support (via Python), got error: %v", err)
			}
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
