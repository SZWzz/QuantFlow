// Package normalize provides unified data types and field mappers for normalizing
// data from various sources (EastMoney, Binance, IBKR, etc.) into canonical formats.
package normalize

// OHLCVBar is the canonical OHLCV data type used across the entire system.
// All adapters SHOULD normalize their output to this type.
type OHLCVBar struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"` // "2006-01-02"
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}
