package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quantflow/internal/market"
)

// commodityDef maps a label to its Sina futures symbol.
type commodityDef struct {
	label  string // "CL" → WTI crude, "NG" → natural gas
	symbol string // "hf_CL", "hf_NG"
	name   string
	nameCN string
	unit   string
}

var commoditySymbols = []commodityDef{
	{symbol: "hf_CL", name: "WTI Crude Oil", nameCN: "WTI原油", unit: "$/bbl"},
	{symbol: "hf_NG", name: "Natural Gas", nameCN: "天然气", unit: "$/MMBtu"},
}

// queryCommodityQuotes fetches real-time futures prices from Sina Finance.
// Returns a map with "commodities" array and "updated_at" timestamp.
func queryCommodityQuotes(reg *market.AdapterRegistry) map[string]interface{} {
	client := &http.Client{Timeout: 5 * time.Second}
	var symbols []string
	for _, c := range commoditySymbols {
		symbols = append(symbols, c.symbol)
	}
	url := "http://hq.sinajs.cn/list=" + strings.Join(symbols, ",")

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn/")

	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("commodity quotes: http error", "error", err)
		return map[string]interface{}{"commodities": []interface{}{}, "error": err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	results := parseSinaCommodities(string(body))
	return map[string]interface{}{
		"commodities": results,
		"updated_at":  time.Now().Unix(),
	}
}

func parseSinaCommodities(body string) []map[string]interface{} {
	var results []map[string]interface{}
	parts := strings.Split(body, ";")
	for i, part := range parts {
		if i >= len(commoditySymbols) {
			break
		}
		cfg := commoditySymbols[i]
		// Format: var hq_str_hf_CL="price,,bid,ask,high,low,time,..."
		start := 0
		for start < len(part) && part[start] != '"' {
			start++
		}
		if start >= len(part) {
			continue
		}
		start++
		end := start
		for end < len(part) && part[end] != '"' {
			end++
		}
		content := part[start:end]
		fields := strings.Split(content, ",")
		if len(fields) < 7 {
			continue
		}

		price, _ := strconv.ParseFloat(fields[0], 64)
		open, _ := strconv.ParseFloat(fields[2], 64)
		high, _ := strconv.ParseFloat(fields[4], 64)
		low, _ := strconv.ParseFloat(fields[5], 64)
		prevClose, _ := strconv.ParseFloat(fields[8], 64)
		changePct := 0.0
		if prevClose > 0 {
			changePct = (price - prevClose) / prevClose * 100
		}

		results = append(results, map[string]interface{}{
			"symbol":      cfg.symbol,
			"name":        cfg.name,
			"name_cn":     cfg.nameCN,
			"price":       price,
			"open":        open,
			"high":        high,
			"low":         low,
			"change_pct":  changePct,
			"unit":        cfg.unit,
			"updated":     fields[6],
		})
	}
	_ = json.NewDecoder // ensure json imported
	return results
}

// ensure json and fmt are used
var _ = fmt.Sprintf
