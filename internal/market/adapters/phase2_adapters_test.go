package adapters

import (
	"context"
	"testing"
	"time"
)

// ── THS Hot Stocks ────────────────────────────────────────────────────

func TestTHSHotAdapter_Interface(t *testing.T) {
	a := NewTHSHotAdapter()
	if a.Name() != "ths_hot" {
		t.Errorf("expected name 'ths_hot', got %s", a.Name())
	}
}

func TestTHSHotAdapter_FetchHotStocks(t *testing.T) {
	a := NewTHSHotAdapter()
	stocks, err := a.FetchHotStocks(context.Background(), "")
	if err != nil {
		t.Logf("THS hot stocks fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d hot stocks", len(stocks))
	if len(stocks) > 0 {
		s := stocks[0]
		if s.Code == "" {
			t.Error("stock code should not be empty")
		}
		t.Logf("first: %s %s reason=%s change=%.2f%%", s.Code, s.Name, s.Reason, s.ChangePct)
	}
}

// ── THS Northbound ────────────────────────────────────────────────────

func TestTHSNorthboundAdapter_Interface(t *testing.T) {
	a := NewTHSNorthboundAdapter()
	if a.Name() != "ths_northbound" {
		t.Errorf("expected name 'ths_northbound', got %s", a.Name())
	}
}

func TestTHSNorthboundAdapter_FetchMinuteFlow(t *testing.T) {
	a := NewTHSNorthboundAdapter()
	points, err := a.FetchMinuteFlow(context.Background())
	if err != nil {
		t.Logf("northbound fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d minute points", len(points))
}

func TestTHSNorthboundAdapter_GetHistory(t *testing.T) {
	a := NewTHSNorthboundAdapter()
	snapshots, err := a.GetHistory(5)
	if err != nil {
		t.Logf("history fetch error: %v", err)
		return
	}
	t.Logf("cached %d daily snapshots", len(snapshots))
}

// ── EastMoney Concept ─────────────────────────────────────────────────

func TestEastMoneyConceptAdapter_Interface(t *testing.T) {
	a := NewEastMoneyConceptAdapter()
	if a.Name() != "eastmoney_concept" {
		t.Errorf("expected name 'eastmoney_concept', got %s", a.Name())
	}
}

func TestEastMoneyConceptAdapter_FetchBlocks(t *testing.T) {
	a := NewEastMoneyConceptAdapter()
	blocks, err := a.FetchConceptBlocks(context.Background(), "600519")
	if err != nil {
		t.Logf("concept blocks fetch failed (network): %v", err)
		return
	}
	t.Logf("600519 belongs to %d blocks", len(blocks))
	for i, b := range blocks {
		if i >= 5 {
			break
		}
		t.Logf("  %s (%s) chg=%.2f%% leader=%s", b.Name, b.Code, b.ChangePct, b.LeadStock)
	}
}

// ── EastMoney FundFlow ────────────────────────────────────────────────

func TestEastMoneyFundFlowAdapter_FetchMinute(t *testing.T) {
	a := NewEastMoneyFundFlowAdapter()
	flows, err := a.FetchMinuteFlow(context.Background(), "600519")
	if err != nil {
		t.Logf("fund flow minute fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d minute flow points", len(flows))
}

func TestEastMoneyFundFlowAdapter_FetchDaily(t *testing.T) {
	a := NewEastMoneyFundFlowAdapter()
	flows, err := a.FetchDailyFlow(context.Background(), "600519")
	if err != nil {
		t.Logf("fund flow daily fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d daily flow records", len(flows))
}

// ── EastMoney Signals ─────────────────────────────────────────────────

func TestEastMoneySignalsAdapter_IndustryRanks(t *testing.T) {
	a := NewEastMoneySignalsAdapter()
	ranks, err := a.FetchIndustryRanks(context.Background(), "CN", 10)
	if err != nil {
		t.Logf("industry ranks fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d industry ranks", len(ranks))
	for i, r := range ranks {
		if i >= 3 {
			break
		}
		t.Logf("  #%d %s: %.2f%% up=%d down=%d leader=%s",
			r.Rank, r.Name, r.ChangePct, r.UpCount, r.DownCount, r.Leader)
	}
}

func TestEastMoneySignalsAdapter_DailyDragonTiger(t *testing.T) {
	a := NewEastMoneySignalsAdapter()
	date := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	stocks, err := a.FetchDailyDragonTiger(context.Background(), date, 0)
	if err != nil {
		t.Logf("dragon tiger fetch failed (network/non-trading-day): %v", err)
		return
	}
	t.Logf("dragon tiger board for %s: %d stocks", date, len(stocks))
}

func TestEastMoneySignalsAdapter_LockupExpiry(t *testing.T) {
	a := NewEastMoneySignalsAdapter()
	events, err := a.FetchLockupExpiry(context.Background(), "600519")
	if err != nil {
		t.Logf("lockup expiry fetch failed (network): %v", err)
		return
	}
	t.Logf("lockup events: %d", len(events))
}

// ── EastMoney Capital ─────────────────────────────────────────────────

func TestEastMoneyCapitalAdapter_MarginTrading(t *testing.T) {
	a := NewEastMoneyCapitalAdapter()
	data, err := a.FetchMarginTrading(context.Background(), "600519", 5)
	if err != nil {
		t.Logf("margin trading fetch failed (network): %v", err)
		return
	}
	t.Logf("margin trading records: %d", len(data))
}

func TestEastMoneyCapitalAdapter_BlockTrades(t *testing.T) {
	a := NewEastMoneyCapitalAdapter()
	data, err := a.FetchBlockTrades(context.Background(), "600519", 5)
	if err != nil {
		t.Logf("block trades fetch failed (network): %v", err)
		return
	}
	t.Logf("block trade records: %d", len(data))
}

func TestEastMoneyCapitalAdapter_HolderChanges(t *testing.T) {
	a := NewEastMoneyCapitalAdapter()
	data, err := a.FetchHolderChanges(context.Background(), "600519", 5)
	if err != nil {
		t.Logf("holder changes fetch failed (network): %v", err)
		return
	}
	t.Logf("holder change records: %d", len(data))
}

func TestEastMoneyCapitalAdapter_DividendHistory(t *testing.T) {
	a := NewEastMoneyCapitalAdapter()
	data, err := a.FetchDividendHistory(context.Background(), "600519", 5)
	if err != nil {
		t.Logf("dividend fetch failed (network): %v", err)
		return
	}
	t.Logf("dividend records: %d", len(data))
}

// ── Sina Financials ───────────────────────────────────────────────────

func TestSinaFinancialsAdapter_IncomeStatement(t *testing.T) {
	a := NewSinaFinancialsAdapter()
	periods, err := a.FetchIncomeStatement(context.Background(), "600519", 4)
	if err != nil {
		t.Logf("income statement fetch failed (network): %v", err)
		return
	}
	t.Logf("income statement periods: %d", len(periods))
	for _, p := range periods {
		t.Logf("  period=%s items=%d", p.Period, len(p.Items))
	}
}

func TestSinaFinancialsAdapter_BalanceSheet(t *testing.T) {
	a := NewSinaFinancialsAdapter()
	periods, err := a.FetchBalanceSheet(context.Background(), "600519", 2)
	if err != nil {
		t.Logf("balance sheet fetch failed (network): %v", err)
		return
	}
	t.Logf("balance sheet periods: %d", len(periods))
}
