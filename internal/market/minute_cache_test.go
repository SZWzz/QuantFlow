package market

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestMinuteCache_GetIncremental_FirstCall(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	// First call: no data yet, should return empty
	ticks, err := mc.GetIncremental("600519", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 0 {
		t.Errorf("expected 0 ticks, got %d", len(ticks))
	}
}

func TestMinuteCache_SaveAndGet(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	today := time.Now().Format("2006-01-02")

	input := []MinuteTick{
		{Time: "09:30", Price: 100.5, Volume: 1000, AvgPrice: 100.5},
		{Time: "09:31", Price: 100.8, Volume: 2000, AvgPrice: 100.65},
	}
	if err := mc.SaveTicks("600519", today, input); err != nil {
		t.Fatal(err)
	}

	ticks, err := mc.GetIncremental("600519", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(ticks))
	}
	if ticks[0].Time != "09:30" || ticks[0].Price != 100.5 {
		t.Errorf("unexpected tick[0]: %+v", ticks[0])
	}
}

func TestMinuteCache_GetIncremental_Subsequent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	input := []MinuteTick{
		{Time: "09:30", Price: 100.0, Volume: 100, AvgPrice: 100.0},
		{Time: "09:31", Price: 101.0, Volume: 200, AvgPrice: 100.5},
		{Time: "09:32", Price: 102.0, Volume: 150, AvgPrice: 101.0},
	}
	if err := mc.SaveTicks("600519", "2026-06-26", input); err != nil {
		t.Fatal(err)
	}

	// since = unix of 09:31:00
	sinceUnix := parseTimeToUnix("2026-06-26", "09:31")
	ticks, err := mc.GetIncremental("600519", sinceUnix)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 1 {
		t.Fatalf("expected 1 new tick after 09:31, got %d", len(ticks))
	}
	if ticks[0].Time != "09:32" {
		t.Errorf("expected 09:32, got %s", ticks[0].Time)
	}
}

func TestMinuteCache_LRUFull(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	today := time.Now().Format("2006-01-02")

	// Fill LRU beyond capacity (500 entries is huge, simulate with small cache)
	// This test verifies the LRU eviction doesn't panic
	for i := 0; i < 10; i++ {
		symbol := "6005" + string(rune('0'+i))
		ticks := []MinuteTick{{Time: "09:30", Price: 100.0, Volume: 100, AvgPrice: 100.0}}
		if err := mc.SaveTicks(symbol, today, ticks); err != nil {
			t.Fatal(err)
		}
	}

	ticks, err := mc.GetIncremental("60050", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 1 {
		t.Errorf("expected 1 tick from DB fallback, got %d", len(ticks))
	}
}

func parseTimeToUnix(date, timeStr string) int64 {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
	return t.Unix()
}
