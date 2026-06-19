package market

import "strings"

// NormalizeInterval standardizes the interval string to a canonical form.
// Frontend sends lowercase ("1d"), but many adapters expect uppercase ("1D").
// This function provides a single normalization point called at the registry
// entry point before dispatching to individual adapters.
//
// Mapping:
//
//	5m, 15m, 30m, 1h, 4h → kept lowercase (used by crypto/minute adapters)
//	1d, 1D → "1D" (daily)
//	1w, 1W → "1W" (weekly)
//	1M, 1month → "1M" (monthly)
func NormalizeInterval(interval string) string {
	// Exact/preferred case matching first — handles the "1M" (month) vs "1m" (minute)
	// ambiguity by checking the original input before lowering.
	switch interval {
	case "1D", "1d", "day", "daily":
		return "1D"
	case "1W", "1w", "week", "weekly":
		return "1W"
	case "1M", "1month", "monthly":
		return "1M"
	}

	// Lowered matching for minute-level intervals — safe because "1M" (month)
	// was already captured above.
	switch strings.ToLower(interval) {
	case "1m", "1min":
		return "1m"
	case "5m", "5min":
		return "5m"
	case "15m", "15min":
		return "15m"
	case "30m", "30min":
		return "30m"
	case "1h", "1hour", "hourly":
		return "1h"
	case "4h", "4hour":
		return "4h"
	default:
		return interval // pass through unknown values
	}
}
