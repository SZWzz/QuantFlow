// stock_list_embedded.go — embedded fallback stock lists for all markets.
// push2.eastmoney.com is unreliable; Sina Finance's Market Center API
// provides the CN list which is pre-fetched at build time. Top HK/US
// stocks are curated manually as fallback when the API is unreachable.
package market

import (
	_ "embed"
	"encoding/json"
	"log/slog"
)

//go:embed stock_list_cn.json
var cnStockListJSON []byte

//go:embed stock_list_hk.json
var hkStockListJSON []byte

//go:embed stock_list_us.json
var usStockListJSON []byte

type embeddedStockEntry struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Market string `json:"market"`
}

// loadEmbeddedCNStockList reads the pre-fetched A-share list from the
// embedded JSON. Returns ~5500 entries with computed pinyin abbreviations.
func loadEmbeddedCNStockList() []StockEntry {
	return loadEmbeddedList(cnStockListJSON, "CN")
}

// loadEmbeddedHKStockList reads the curated HK top-50 stock list.
func loadEmbeddedHKStockList() []StockEntry {
	return loadEmbeddedList(hkStockListJSON, "HK")
}

// loadEmbeddedUSStockList reads the curated US top-50 stock list.
func loadEmbeddedUSStockList() []StockEntry {
	return loadEmbeddedList(usStockListJSON, "US")
}

func loadEmbeddedList(data []byte, market string) []StockEntry {
	var raw []embeddedStockEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		slog.Error("symbol_search: failed to parse embedded stock list", "market", market, "error", err)
		return nil
	}
	entries := make([]StockEntry, 0, len(raw))
	for _, r := range raw {
		entries = append(entries, StockEntry{
			Code:   r.Code,
			Name:   r.Name,
			Market: r.Market,
			Pinyin: pinyinAbbr(r.Name),
		})
	}
	slog.Info("symbol_search: stock list loaded from embedded data", "market", market, "stocks", len(entries))
	return entries
}
