// stock_list_embedded.go — embedded fallback for CN A-share stock list.
// push2.eastmoney.com is unreliable; Sina Finance's Market Center API
// provides the full list which is pre-fetched at build time.
package market

import (
	_ "embed"
	"encoding/json"
	"log/slog"
)

//go:embed stock_list_cn.json
var cnStockListJSON []byte

type cnStockEntry struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Market string `json:"market"`
}

// loadEmbeddedCNStockList reads the pre-fetched A-share list from the
// embedded JSON. Returns ~5500 entries with computed pinyin abbreviations.
func loadEmbeddedCNStockList() []StockEntry {
	var raw []cnStockEntry
	if err := json.Unmarshal(cnStockListJSON, &raw); err != nil {
		slog.Error("symbol_search: failed to parse embedded CN stock list", "error", err)
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
	slog.Info("symbol_search: CN stock list loaded from embedded data", "stocks", len(entries))
	return entries
}
