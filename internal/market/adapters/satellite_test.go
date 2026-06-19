// internal/market/adapters/satellite_test.go
package adapters

import (
	"context"
	"testing"
	"time"
)

func TestSatelliteAdapter_Name(t *testing.T) {
	a := NewSatelliteAdapter()
	if a.Name() != "satellite" {
		t.Errorf("Name() = %s, want satellite", a.Name())
	}
}

func TestSatelliteAdapter_IsAvailable(t *testing.T) {
	a := NewSatelliteAdapter()
	ctx := context.Background()
	available := a.IsAvailable(ctx)
	t.Logf("IsAvailable=%v", available)
	// Test should not panic regardless of network state
}

func TestSatelliteAdapter_FetchEnergyData(t *testing.T) {
	a := NewSatelliteAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("NASA POWER API not reachable")
	}

	// Test solar data for Gobi Solar Base
	points, err := a.FetchEnergyData(ctx, 40.5, 100.0, "ALLSKY_SFC_SW_DWN")
	if err != nil {
		t.Fatalf("FetchEnergyData (solar) error: %v", err)
	}
	t.Logf("got %d solar energy points for Gobi", len(points))
	if len(points) > 0 {
		t.Logf("first point: date=%s value=%.2f kWh/m^2/day", points[0].Date, points[0].Value)
	}

	// Test wind data for North Sea
	windPoints, err := a.FetchEnergyData(ctx, 56.0, 3.0, "WS2M")
	if err != nil {
		t.Fatalf("FetchEnergyData (wind) error: %v", err)
	}
	t.Logf("got %d wind speed points for North Sea", len(windPoints))
	if len(windPoints) > 0 {
		t.Logf("first point: date=%s value=%.2f m/s", windPoints[0].Date, windPoints[0].Value)
	}
}

func TestSatelliteAdapter_FetchWildfireCount(t *testing.T) {
	a := NewSatelliteAdapter()
	ctx := context.Background()

	count, err := a.FetchWildfireCount(ctx, 7)
	if err != nil {
		t.Fatalf("FetchWildfireCount error: %v", err)
	}
	t.Logf("wildfire count: %d", count)
	// Should return a non-negative count (mock or real)
	if count < 0 {
		t.Errorf("wildfire count should be non-negative, got %d", count)
	}
}

func TestSatelliteRegions(t *testing.T) {
	if len(SatelliteRegions) != 5 {
		t.Errorf("expected 5 satellite regions, got %d", len(SatelliteRegions))
	}

	expectedIDs := []string{"texas", "north-sea", "gobi", "sahara", "midwest"}
	for i, id := range expectedIDs {
		if SatelliteRegions[i].ID != id {
			t.Errorf("region[%d]: expected ID %s, got %s", i, id, SatelliteRegions[i].ID)
		}
		if SatelliteRegions[i].NameCN == "" {
			t.Errorf("region[%d] (%s): NameCN should not be empty", i, id)
		}
		if SatelliteRegions[i].Lat == 0 && SatelliteRegions[i].Lon == 0 {
			t.Errorf("region[%d] (%s): lat/lon should not be zero", i, id)
		}
		if SatelliteRegions[i].AssetLink == "" {
			t.Errorf("region[%d] (%s): AssetLink should not be empty", i, id)
		}
	}

	t.Logf("all 5 satellite regions configured correctly")
}
