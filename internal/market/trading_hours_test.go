package market

import (
	"testing"
	"time"
)

func TestIsTradingHours_CN(t *testing.T) {
	loc := mustLoadLocation("Asia/Shanghai")

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"周一 9:29（盘前）", time.Date(2026, 7, 6, 9, 29, 0, 0, loc), false},
		{"周一 9:30（开盘）", time.Date(2026, 7, 6, 9, 30, 0, 0, loc), true},
		{"周一 11:29（上午盘中）", time.Date(2026, 7, 6, 11, 29, 0, 0, loc), true},
		{"周一 11:30（上午收盘）", time.Date(2026, 7, 6, 11, 30, 0, 0, loc), false},
		{"周一 12:59（午休）", time.Date(2026, 7, 6, 12, 59, 0, 0, loc), false},
		{"周一 13:00（下午开盘）", time.Date(2026, 7, 6, 13, 0, 0, 0, loc), true},
		{"周一 14:59（下午盘中）", time.Date(2026, 7, 6, 14, 59, 0, 0, loc), true},
		{"周一 15:00（收盘）", time.Date(2026, 7, 6, 15, 0, 0, 0, loc), false},
		{"周六任意时间", time.Date(2026, 7, 4, 10, 0, 0, 0, loc), false},
		{"周日任意时间", time.Date(2026, 7, 5, 10, 0, 0, 0, loc), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTradingHours("CN", tt.time)
			if got != tt.want {
				t.Errorf("isTradingHours(CN, %v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestIsTradingHours_HK(t *testing.T) {
	loc := mustLoadLocation("Asia/Hong_Kong")

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"周一 9:29（盘前）", time.Date(2026, 7, 6, 9, 29, 0, 0, loc), false},
		{"周一 9:30（开盘）", time.Date(2026, 7, 6, 9, 30, 0, 0, loc), true},
		{"周一 11:59（上午盘中）", time.Date(2026, 7, 6, 11, 59, 0, 0, loc), true},
		{"周一 12:00（上午收盘）", time.Date(2026, 7, 6, 12, 0, 0, 0, loc), false},
		{"周一 13:00（下午开盘）", time.Date(2026, 7, 6, 13, 0, 0, 0, loc), true},
		{"周一 15:59（下午盘中）", time.Date(2026, 7, 6, 15, 59, 0, 0, loc), true},
		{"周一 16:00（收盘）", time.Date(2026, 7, 6, 16, 0, 0, 0, loc), false},
		{"周六任意时间", time.Date(2026, 7, 4, 10, 0, 0, 0, loc), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTradingHours("HK", tt.time)
			if got != tt.want {
				t.Errorf("isTradingHours(HK, %v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestIsTradingHours_US(t *testing.T) {
	loc := mustLoadLocation("America/New_York")

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"周一 3:59（盘前）", time.Date(2026, 7, 6, 3, 59, 0, 0, loc), false},
		{"周一 4:00（盘前开始）", time.Date(2026, 7, 6, 4, 0, 0, 0, loc), true},
		{"周一 9:29（盘前）", time.Date(2026, 7, 6, 9, 29, 0, 0, loc), true},
		{"周一 9:30（开盘）", time.Date(2026, 7, 6, 9, 30, 0, 0, loc), true},
		{"周一 15:59（盘中）", time.Date(2026, 7, 6, 15, 59, 0, 0, loc), true},
		{"周一 16:00（收盘）", time.Date(2026, 7, 6, 16, 0, 0, 0, loc), false},
		{"周六任意时间", time.Date(2026, 7, 4, 10, 0, 0, 0, loc), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTradingHours("US", tt.time)
			if got != tt.want {
				t.Errorf("isTradingHours(US, %v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestIsTradingHours_CRYPTO(t *testing.T) {
	// Crypto is 24/7 — any time, any day.
	loc := time.UTC
	for _, tt := range []struct {
		name string
		time time.Time
	}{
		{"周一 3:00", time.Date(2026, 7, 6, 3, 0, 0, 0, loc)},
		{"周六 10:00", time.Date(2026, 7, 4, 10, 0, 0, 0, loc)},
		{"周日 23:59", time.Date(2026, 7, 5, 23, 59, 0, 0, loc)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !isTradingHours("CRYPTO", tt.time) {
				t.Errorf("isTradingHours(CRYPTO, %v) = false, want true", tt.time)
			}
		})
	}
}

func TestIsTradingHours_UnknownMarket(t *testing.T) {
	// Unknown market defaults to true (allow).
	loc := time.UTC
	got := isTradingHours("UNKNOWN", time.Date(2026, 7, 4, 10, 0, 0, 0, loc))
	if !got {
		t.Error("isTradingHours(UNKNOWN, Saturday 10:00) = false, want true (unknown market should allow)")
	}
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
