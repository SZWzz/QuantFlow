package market

import "time"

// IsTradingHours reports whether the given market is currently in a trading
// session. Outside trading hours, real-time quote data is not available from
// any adapter, so callers can skip the fetch chain immediately.
//
// Markets: CN, HK, US, CRYPTO. An empty or unknown market defaults to true
// (allow — the adapter chain will decide whether data is available).
//
// Holiday calendars are NOT implemented. On exchange holidays the session
// check returns true but all adapters will naturally fail; this avoids a
// hard dependency on external holiday data files.
func IsTradingHours(market string) bool {
	now := time.Now()
	return isTradingHours(market, now)
}

func isTradingHours(market string, now time.Time) bool {
	switch market {
	case "CN":
		return weekdayOnly(now) && cnTradingHours(now)
	case "HK":
		return weekdayOnly(now) && hkTradingHours(now)
	case "US":
		return weekdayOnly(now) && usTradingHours(now)
	case "CRYPTO":
		return true
	default:
		// Unknown market — allow (caller will discover unavailability naturally).
		return true
	}
}

func weekdayOnly(t time.Time) bool {
	return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday
}

func cnTradingHours(t time.Time) bool {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return true
	}
	t = t.In(loc)
	h, m := t.Hour(), t.Minute()
	minutes := h*60 + m
	// Morning: 09:30 – 11:30 (570 – 690)
	// Afternoon: 13:00 – 15:00 (780 – 900)
	return (minutes >= 570 && minutes < 690) || (minutes >= 780 && minutes < 900)
}

func hkTradingHours(t time.Time) bool {
	loc, err := time.LoadLocation("Asia/Hong_Kong")
	if err != nil {
		return true
	}
	t = t.In(loc)
	h, m := t.Hour(), t.Minute()
	minutes := h*60 + m
	// Morning: 09:30 – 12:00 (570 – 720)
	// Afternoon: 13:00 – 16:00 (780 – 960)
	return (minutes >= 570 && minutes < 720) || (minutes >= 780 && minutes < 960)
}

func usTradingHours(t time.Time) bool {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return true
	}
	t = t.In(loc)
	h, m := t.Hour(), t.Minute()
	minutes := h*60 + m
	// Regular session: 09:30 – 16:00 ET (570 – 960)
	// Pre-market: 04:00 – 09:30 (240 – 570) — allow for early data
	return minutes >= 240 && minutes < 960
}
