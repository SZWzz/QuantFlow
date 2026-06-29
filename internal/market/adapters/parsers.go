package adapters

import (
	"fmt"
	"strings"
	"time"

	"quantflow/internal/market"
)

// parseFloatSafe parses a float64 from a string, returning 0 on failure.
func parseFloatSafe(s string) float64 {
	var f float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}

// toSinaCode converts a symbol to Sina format.
// Delegates to SymbolIdentity for CN A-shares; handles HK and US directly.
func toSinaCode(symbol string) string {
	// HK stocks: "00700" → "hk00700", "00700.HK" → "hk00700"
	if isHKCode(symbol) {
		return toSinaHKCode(symbol)
	}
	// US stocks: any non-CN, non-HK alphanumeric ticker → "gb_aapl"
	if isUSCode(symbol) {
		return toSinaUSCode(symbol)
	}
	// CN A-shares: use SymbolIdentity
	id, err := market.NormalizeCN(symbol)
	if err != nil {
		return strings.ToLower(symbol)
	}
	return id.ToSina()
}

// isUSCode detects US stock symbols (alphanumeric tickers, not CN/HK format).
func isUSCode(symbol string) bool {
	s := strings.TrimSpace(strings.ToUpper(symbol))
	// Strip exchange suffix
	s = strings.TrimSuffix(s, ".US")
	// Not CN (6-digit numeric) and not HK (5-digit numeric)
	if len(s) == 6 && isAllDigitsStr(s) {
		return false
	}
	if len(s) == 5 && isAllDigitsStr(s) {
		return false
	}
	// Must be 1-5 character alphanumeric ticker
	if len(s) < 1 || len(s) > 5 {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func isAllDigitsStr(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isHKCode detects HK stock symbols (5-digit numeric codes, optionally with .HK suffix).
func isHKCode(symbol string) bool {
	s := strings.TrimSuffix(strings.ToUpper(symbol), ".HK")
	if len(s) != 5 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseSinaQuote parses Sina's CSV-like response format.
//
// Example: var hq_str_sh600519="茅台,1950.00,1945.00,..."
func parseSinaQuote(symbol, body string) (*market.QuoteSnapshot, error) {
	idx := strings.Index(body, "\"")
	if idx == -1 {
		return nil, fmt.Errorf("sina: unexpected response format")
	}
	content := body[idx+1:]
	endIdx := strings.LastIndex(content, "\"")
	if endIdx == -1 {
		return nil, fmt.Errorf("sina: unexpected response format")
	}
	content = content[:endIdx]

	fields := strings.Split(content, ",")
	if len(fields) < 32 {
		return nil, fmt.Errorf("sina: insufficient fields: got %d", len(fields))
	}

	// Sina field mapping (1-indexed):
	// [0]=name, [1]=open, [2]=prevClose, [3]=last, [4]=high, [5]=low,
	// [6]=bid(buy1), [7]=ask(sell1), [8]=volume(手), [9]=amount(万)
	open := parseFloatSafe(fields[1])
	last := parseFloatSafe(fields[3])
	high := parseFloatSafe(fields[4])
	low := parseFloatSafe(fields[5])
	bid := parseFloatSafe(fields[6])
	ask := parseFloatSafe(fields[7])
	volume := parseFloatSafe(fields[8]) * 100 // 手→股
	prevClose := parseFloatSafe(fields[2])
	turnover := parseFloatSafe(fields[9]) * 10000 // 万→元

	change := last - prevClose
	changePct := 0.0
	if prevClose > 0 {
		changePct = (change / prevClose) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Name:      cleanName(fields[0]),
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		PrevClose: prevClose,
		Bid:       bid,
		Ask:       ask,
		Volume:    volume,
		Turnover:  turnover,
		Change:    change,
		ChangePct: changePct,
		Exchange:  "CN",
		Timestamp: time.Now().UnixMilli(),
	}, nil
}
