package adapters

import (
	"context"
	"testing"
)

func TestEastMoneyNewsAdapter_Interface(t *testing.T) {
	adapter := NewEastMoneyNewsAdapter()
	if adapter.Name() != "eastmoney_news" {
		t.Errorf("expected name 'eastmoney_news', got %s", adapter.Name())
	}
}

func TestEastMoneyNewsAdapter_FetchStockNews_Mock(t *testing.T) {
	// This test verifies the adapter doesn't panic on invalid input.
	// Real HTTP tests are skipped (require live network + EastMoney API availability).
	adapter := NewEastMoneyNewsAdapter()

	t.Run("zero limit defaults to 5", func(t *testing.T) {
		articles, err := adapter.FetchStockNews(context.Background(), "600519", 0)
		if err != nil {
			t.Logf("fetch failed (expected without network): %v", err)
			return // test is informative, not blocking
		}
		if len(articles) > 5 {
			t.Errorf("expected max 5 articles for limit=0, got %d", len(articles))
		}
	})

	t.Run("limit capped at 50", func(t *testing.T) {
		articles, err := adapter.FetchStockNews(context.Background(), "600519", 100)
		if err != nil {
			t.Logf("fetch failed (expected without network): %v", err)
			return
		}
		if len(articles) > 50 {
			t.Errorf("articles exceeded cap 50, got %d", len(articles))
		}
	})
}

func TestEastMoneyNewsAdapter_IsAvailable(t *testing.T) {
	adapter := NewEastMoneyNewsAdapter()
	available := adapter.IsAvailable(context.Background())
	t.Logf("eastmoney_news available: %v", available)
	// Not asserting true/false — depends on network
}

func TestEastMoneyGlobalNewsAdapter_Interface(t *testing.T) {
	adapter := NewEastMoneyGlobalNewsAdapter()
	if adapter.Name() != "eastmoney_global_news" {
		t.Errorf("expected name 'eastmoney_global_news', got %s", adapter.Name())
	}

	// Verify it satisfies GlobalNewsAdapter interface
	var _ GlobalNewsAdapter = adapter
}

func TestEastMoneyGlobalNewsAdapter_FetchGlobalNews(t *testing.T) {
	adapter := NewEastMoneyGlobalNewsAdapter()
	articles, err := adapter.FetchGlobalNews(context.Background(), 3)
	if err != nil {
		t.Logf("global news fetch failed (expected without network): %v", err)
		return
	}
	t.Logf("fetched %d global news articles", len(articles))
}

func TestNewsArticle_Struct(t *testing.T) {
	a := NewsArticle{
		Symbol:  "600519",
		Title:   "茅台股价创新高",
		Content: "贵州茅台今日收盘价突破2000元...",
		Time:    "2026-06-19",
		Source:  "eastmoney",
		URL:     "https://example.com",
	}
	if a.Symbol != "600519" {
		t.Error("Symbol field mismatch")
	}
	if a.Title == "" {
		t.Error("Title should not be empty")
	}
}
