package normalize

import (
	"fmt"
	"strings"
)

// Mapper defines a generic field mapper from raw data to a normalized type.
type Mapper[T any] interface {
	// Parse converts raw source data into the normalized type.
	Parse(raw map[string]any) (T, error)
	// Source returns the adapter/source name.
	Source() string
}

// ── Order status mapping ────────────────────────────────────────────

// OrderStatusMapper normalizes broker-specific order status strings.
type OrderStatusMapper struct {
	broker string
}

// NewOrderStatusMapper creates a mapper for the given broker.
func NewOrderStatusMapper(broker string) *OrderStatusMapper {
	return &OrderStatusMapper{broker: broker}
}

// Map converts a broker-specific order status to a canonical string.
// Canonical values: "pending", "filled", "cancelled", "rejected", "partial".
func (m *OrderStatusMapper) Map(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch m.broker {
	case "ibkr":
		switch s {
		case "submitted", "presubmitted", "inactive":
			return "pending"
		case "filled":
			return "filled"
		case "cancelled", "apicancelled":
			return "cancelled"
		default:
			return "pending"
		}
	case "binance":
		switch s {
		case "new", "partially_filled":
			return "pending"
		case "filled":
			return "filled"
		case "canceled", "expired", "rejected":
			return "cancelled"
		default:
			return "pending"
		}
	case "alpaca":
		switch s {
		case "accepted", "new", "partially_filled", "held":
			return "pending"
		case "filled":
			return "filled"
		case "canceled", "expired", "rejected", "suspended":
			return "cancelled"
		default:
			return "pending"
		}
	default:
		return s
	}
}

// Source returns the broker name.
func (m *OrderStatusMapper) Source() string { return m.broker }

// ── Order type mapping ─────────────────────────────────────────────

// OrderTypeMapper normalizes broker-specific order type strings.
type OrderTypeMapper struct {
	broker string
}

// NewOrderTypeMapper creates a mapper for the given broker.
func NewOrderTypeMapper(broker string) *OrderTypeMapper {
	return &OrderTypeMapper{broker: broker}
}

// Map converts a broker-specific order type to a canonical string.
// Canonical values: "market", "limit", "stop", "stop_limit".
func (m *OrderTypeMapper) Map(orderType string) string {
	s := strings.ToUpper(strings.TrimSpace(orderType))
	switch m.broker {
	case "ibkr":
		switch s {
		case "MKT":
			return "market"
		case "LMT":
			return "limit"
		case "STP":
			return "stop"
		default:
			return "market"
		}
	case "binance":
		switch s {
		case "MARKET":
			return "market"
		case "LIMIT", "LIMIT_MAKER":
			return "limit"
		case "STOP_LOSS_LIMIT", "STOP_LOSS", "TAKE_PROFIT", "TAKE_PROFIT_LIMIT":
			return "stop"
		default:
			return "market"
		}
	case "alpaca":
		switch s {
		case "MARKET":
			return "market"
		case "LIMIT":
			return "limit"
		case "STOP", "STOP_LIMIT":
			return "stop"
		default:
			return "market"
		}
	default:
		return s
	}
}

// Source returns the broker name.
func (m *OrderTypeMapper) Source() string { return m.broker }

// ── OHLCV Mapper ───────────────────────────────────────────────────

// OHLCVMapper maps a raw map (e.g. from CSV, JSON) to a normalized OHLCVBar.
type OHLCVMapper struct {
	source  string
	columns map[string]string // canonical -> raw field name
}

// NewOHLCVMapper creates an OHLCV mapper with field name mappings.
// columns maps canonical field names ("symbol", "open", etc.) to raw source field names.
func NewOHLCVMapper(source string, columns map[string]string) *OHLCVMapper {
	return &OHLCVMapper{source: source, columns: columns}
}

// Parse converts a raw data map to a normalized OHLCVBar.
func (m *OHLCVMapper) Parse(raw map[string]any) (OHLCVBar, error) {
	var bar OHLCVBar
	var errs []string

	if v, ok := raw[m.columns["symbol"]]; ok {
		bar.Symbol = fmt.Sprintf("%v", v)
	} else {
		errs = append(errs, "symbol")
	}
	if v, ok := raw[m.columns["date"]]; ok {
		bar.Date = fmt.Sprintf("%v", v)
	}
	if v, ok := raw[m.columns["open"]]; ok {
		bar.Open = toFloat64(v)
	}
	if v, ok := raw[m.columns["high"]]; ok {
		bar.High = toFloat64(v)
	}
	if v, ok := raw[m.columns["low"]]; ok {
		bar.Low = toFloat64(v)
	}
	if v, ok := raw[m.columns["close"]]; ok {
		bar.Close = toFloat64(v)
	}
	if v, ok := raw[m.columns["volume"]]; ok {
		bar.Volume = NormalizeVolume(m.source, toFloat64(v))
	}

	if len(errs) > 0 {
		return bar, fmt.Errorf("OHLCVMapper: missing required fields: %s", strings.Join(errs, ", "))
	}
	return bar, nil
}

func (m *OHLCVMapper) Source() string { return m.source }

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case float32:
		return float64(n)
	case string:
		var f float64
		// Best-effort parse: malformed input degrades to 0 rather than failing the mapping
		_, _ = fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}
