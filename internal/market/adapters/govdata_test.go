package adapters

import (
	"context"
	"os"
	"testing"
)

func TestGovDataAdapter_Name(t *testing.T) {
	adpt := NewGovDataAdapter()
	if adpt.Name() != "govdata" {
		t.Errorf("expected name 'govdata', got %s", adpt.Name())
	}
}

func TestGovDataAdapter_IsAvailable(t *testing.T) {
	adpt := NewGovDataAdapter()
	ctx := context.Background()

	key := os.Getenv("FRED_API_KEY")
	if key == "" {
		t.Skip("FRED_API_KEY not set, skipping IsAvailable test (will return false)")
	}

	available := adpt.IsAvailable(ctx)
	if !available {
		t.Log("FRED_API_KEY is set but FRED API is unreachable (offline or rate limited)")
	}
}

func TestGovDataAdapter_IsAvailable_NoKey(t *testing.T) {
	// Temporarily unset FRED_API_KEY to test the no-key degradation path
	oldKey := os.Getenv("FRED_API_KEY")
	os.Unsetenv("FRED_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("FRED_API_KEY", oldKey)
		}
	}()

	adpt := &GovDataHTTPAdapter{
		indicators: FREDIndicators,
	}
	adpt.apiKey = "" // force empty
	ctx := context.Background()

	if adpt.IsAvailable(ctx) {
		t.Error("expected IsAvailable to return false when FRED_API_KEY is not set")
	}
}

func TestGovDataAdapter_FetchIndicator_NoKey(t *testing.T) {
	adpt := NewGovDataAdapter()
	if os.Getenv("FRED_API_KEY") != "" {
		t.Skip("FRED_API_KEY is set, skipping no-key error test")
	}

	ctx := context.Background()
	_, err := adpt.FetchIndicator(ctx, "GDP", 12)
	if err == nil {
		t.Error("expected error when FRED_API_KEY is not set")
	}
}

func TestGovDataAdapter_FetchIndicator(t *testing.T) {
	key := os.Getenv("FRED_API_KEY")
	if key == "" {
		t.Skip("FRED_API_KEY not set, skipping live FetchIndicator test")
	}

	adpt := NewGovDataAdapter()
	ctx := context.Background()

	points, err := adpt.FetchIndicator(ctx, "UNRATE", 6)
	if err != nil {
		t.Fatalf("FetchIndicator failed: %v", err)
	}
	if len(points) == 0 {
		t.Error("expected at least one data point for UNRATE")
	}
	t.Logf("got %d points for UNRATE, latest: date=%s value=%.2f",
		len(points), points[len(points)-1].Date, points[len(points)-1].Value)
}

func TestGovDataAdapter_FetchIndicator_MissingData(t *testing.T) {
	key := os.Getenv("FRED_API_KEY")
	if key == "" {
		t.Skip("FRED_API_KEY not set, skipping live test")
	}

	adpt := NewGovDataAdapter()
	ctx := context.Background()

	// GDPRPI is less commonly available; some dates may have "."
	points, err := adpt.FetchIndicator(ctx, "GDP", 3)
	if err != nil {
		t.Fatalf("FetchIndicator failed: %v", err)
	}
	if len(points) == 0 {
		t.Skip("no data points returned (all may have been '.')")
	}
	if len(points) > 3 {
		t.Errorf("expected at most 3 points, got %d", len(points))
	}
}

func TestGovDataAdapter_FetchCompanyFilings(t *testing.T) {
	adpt := NewGovDataAdapter()
	ctx := context.Background()

	// Apple's CIK: 0000320193
	filings, err := adpt.FetchCompanyFilings(ctx, "0000320193", 5)
	if err != nil {
		t.Skipf("SEC EDGAR API unavailable, skipping: %v", err)
	}
	if len(filings) == 0 {
		t.Error("expected at least one filing for AAPL (CIK 0000320193)")
	}
	t.Logf("got %d filings for AAPL", len(filings))
	for _, f := range filings {
		t.Logf("  %s - %s (%s): %s", f.FilingDate, f.FormType, f.Company, f.URL)
	}
}

func TestFREDIndicators_Count(t *testing.T) {
	if len(FREDIndicators) != 15 {
		t.Errorf("expected 15 FRED indicators, got %d", len(FREDIndicators))
	}
}

func TestFREDIndicators_HasKeyCategories(t *testing.T) {
	categories := map[string]int{}
	for _, meta := range FREDIndicators {
		categories[meta.Category]++
	}

	expectedCats := []string{"gdp", "inflation", "employment", "rates", "energy", "housing"}
	for _, cat := range expectedCats {
		if _, ok := categories[cat]; !ok {
			t.Errorf("expected category %q in FRED indicators", cat)
		}
	}
}

func TestFREDIndicators_ChineseNames(t *testing.T) {
	for id, meta := range FREDIndicators {
		if meta.NameCN == "" {
			t.Errorf("indicator %s has no Chinese name", id)
		}
	}
}
