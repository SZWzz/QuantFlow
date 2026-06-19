package adapters

import (
	"context"
	"testing"
)

// ── EastMoney Report ──────────────────────────────────────────────────

func TestEastMoneyReportAdapter_Interface(t *testing.T) {
	a := NewEastMoneyReportAdapter()
	if a.Name() != "eastmoney_report" {
		t.Errorf("expected name 'eastmoney_report', got %s", a.Name())
	}
}

func TestEastMoneyReportAdapter_FetchReports(t *testing.T) {
	a := NewEastMoneyReportAdapter()
	reports, err := a.FetchReports(context.Background(), "600519", 1)
	if err != nil {
		t.Logf("report fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d reports for 600519", len(reports))
	if len(reports) > 0 {
		r := reports[0]
		t.Logf("first: %s | %s | rating=%s | EPS_this=%.2f EPS_next=%.2f",
			r.PublishDate[:10], r.OrgName, r.Rating,
			r.PredictThisYearEPS, r.PredictNextYearEPS)
	}
}

func TestEastMoneyReportAdapter_FetchConsensusEPS(t *testing.T) {
	a := NewEastMoneyReportAdapter()
	thisYear, nextYear, count, err := a.FetchConsensusEPS(context.Background(), "600519")
	if err != nil {
		t.Logf("consensus EPS fetch failed (network): %v", err)
		return
	}
	t.Logf("consensus from %d reports: thisYear=%.2f nextYear=%.2f", count, thisYear, nextYear)
}

// ── THS Consensus ─────────────────────────────────────────────────────

func TestTHSConsensusAdapter_Interface(t *testing.T) {
	a := NewTHSConsensusAdapter()
	if a.Name() != "ths_consensus" {
		t.Errorf("expected name 'ths_consensus', got %s", a.Name())
	}
}

func TestTHSConsensusAdapter_FetchConsensus(t *testing.T) {
	a := NewTHSConsensusAdapter()
	eps, err := a.FetchConsensus(context.Background(), "600519")
	if err != nil {
		t.Logf("THS consensus fetch failed (network): %v", err)
		return
	}
	if eps == nil {
		t.Log("no consensus data (stock may have no analyst coverage)")
		return
	}
	t.Logf("THS consensus: %d forecast years", len(eps))
	for _, e := range eps {
		t.Logf("  %s: %d analysts, avg=%.2f (min=%.2f max=%.2f)",
			e.Year, e.AnalystCount, e.AvgEPS, e.MinEPS, e.MaxEPS)
	}
}

func TestTHSConsensusAdapter_NoCoverage(t *testing.T) {
	// A small stock likely has no analyst coverage
	a := NewTHSConsensusAdapter()
	eps, err := a.FetchConsensus(context.Background(), "002075")
	if err != nil {
		t.Logf("THS consensus fetch failed (network): %v", err)
		return
	}
	t.Logf("002075 consensus results: %d (expected 0 for small-cap)", len(eps))
}

// ── Cninfo Announcements ──────────────────────────────────────────────

func TestCninfoAdapter_Interface(t *testing.T) {
	a := NewCninfoAdapter()
	if a.Name() != "cninfo" {
		t.Errorf("expected name 'cninfo', got %s", a.Name())
	}
}

func TestCninfoAdapter_FetchAnnouncements(t *testing.T) {
	a := NewCninfoAdapter()
	announcements, err := a.FetchAnnouncements(context.Background(), "600519", 10)
	if err != nil {
		t.Logf("cninfo fetch failed (network): %v", err)
		return
	}
	t.Logf("fetched %d announcements for 600519", len(announcements))
	for i, ann := range announcements {
		if i >= 5 {
			break
		}
		t.Logf("  %s | %s | %s", ann.Date, ann.Type, ann.Title[:min(40, len(ann.Title))])
	}
}

func TestCninfoAdapter_OrgIDLookup(t *testing.T) {
	a := NewCninfoAdapter()
	// Test code with known non-standard orgId (601318 → different format)
	announcements, err := a.FetchAnnouncements(context.Background(), "601318", 5)
	if err != nil {
		t.Logf("cninfo fetch for 601318 failed (network): %v", err)
		return
	}
	t.Logf("601318 announcements: %d (orgId lookup test)", len(announcements))
}
