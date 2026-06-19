package adapters

import (
	"context"
	"testing"
)

func TestEastMoneyGlobalNewsAdapter(t *testing.T) {
	adapter := NewEastMoneyGlobalNewsAdapter()

	t.Run("FetchStockNews filters by symbol", func(t *testing.T) {
		articles, err := adapter.FetchStockNews(context.Background(), "600519", 3)
		if err != nil {
			t.Logf("fetch failed (expected without network): %v", err)
			return
		}
		t.Logf("found %d articles for 600519", len(articles))
		for _, a := range articles {
			if a.Symbol != "600519" && a.Symbol != "" {
				t.Errorf("unexpected symbol in article: %s", a.Symbol)
			}
		}
	})

	t.Run("IsAvailable check", func(t *testing.T) {
		available := adapter.IsAvailable(context.Background())
		t.Logf("eastmoney_global_news available: %v", available)
	})
}
