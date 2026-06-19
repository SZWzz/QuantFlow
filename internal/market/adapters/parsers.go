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

// toSinaCode converts a symbol to Sina format. Delegates to the unified SymbolIdentity.
func toSinaCode(symbol string) string {
	id, err := market.NormalizeCN(symbol)
	if err != nil {
		return strings.ToLower(symbol) // best-effort fallback
	}
	return id.ToSina()
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

	change := last - prevClose
	changePct := 0.0
	if prevClose > 0 {
		changePct = (change / prevClose) * 100
	}

	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		Bid:       bid,
		Ask:       ask,
		Volume:    volume,
		Change:    change,
		ChangePct: changePct,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}
