package nodes

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"quantflow/internal/market"
	"quantflow/internal/workflow"
	"strconv"
	"time"
)

// DataLoaderNode loads market data from various sources.
// Supports csv (local file), mock (synthetic data), and auto (real market data via
// AdapterRegistry fallback chains: EastMoney/Yahoo/Binance etc.).
type DataLoaderNode struct {
	id     string
	params map[string]any
}

func NewDataLoaderNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &DataLoaderNode{id: id, params: params}, nil
}

func (n *DataLoaderNode) ID() string       { return n.id }
func (n *DataLoaderNode) NodeType() string { return "data_loader" }
func (n *DataLoaderNode) Category() string { return "data" }

func (n *DataLoaderNode) InputPorts() []workflow.PortDefinition { return nil }

func (n *DataLoaderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv", Type: workflow.PortOHLCV, Required: false},
		{Name: "close", Type: workflow.PortSeries, Required: false},
		{Name: "open", Type: workflow.PortSeries, Required: false},
		{Name: "high", Type: workflow.PortSeries, Required: false},
		{Name: "low", Type: workflow.PortSeries, Required: false},
		{Name: "volume", Type: workflow.PortSeries, Required: false},
	}
}

func (n *DataLoaderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "source", Type: "string", Default: "auto", Description: "Data source: auto, csv, mock, eastmoney, yahoo, binance"},
		{Name: "path", Type: "string", Default: "", Description: "Path to CSV file (csv source only)"},
		{Name: "symbol", Type: "string", Default: "000300", Description: "Stock symbol (e.g. 000300, AAPL, BTCUSDT)"},
		{Name: "interval", Type: "string", Default: "1d", Description: "Bar interval: 1d, 1w, 1m, 5m, 60m"},
		{Name: "start_date", Type: "string", Default: "", Description: "Start date YYYY-MM-DD (auto: 1 year ago)"},
		{Name: "end_date", Type: "string", Default: "", Description: "End date YYYY-MM-DD (auto: today)"},
	}
}

func parseDate(s string, fallback time.Time) time.Time {
	if s == "" {
		return fallback
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fallback
	}
	return t
}

func (n *DataLoaderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	source := "auto"
	path := ""
	symbol := "000300"
	interval := "1d"
	startDate := ""
	endDate := ""

	if s, ok := n.params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if p, ok := n.params["path"]; ok {
		path = fmt.Sprint(p)
	}
	if s, ok := n.params["symbol"]; ok {
		symbol = fmt.Sprint(s)
	}
	if i, ok := n.params["interval"]; ok {
		interval = fmt.Sprint(i)
	}
	if s, ok := n.params["start_date"]; ok {
		startDate = fmt.Sprint(s)
	}
	if e, ok := n.params["end_date"]; ok {
		endDate = fmt.Sprint(e)
	}
	if s, ok := params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if p, ok := params["path"]; ok {
		path = fmt.Sprint(p)
	}
	if s, ok := params["symbol"]; ok {
		symbol = fmt.Sprint(s)
	}
	if i, ok := params["interval"]; ok {
		interval = fmt.Sprint(i)
	}
	if s, ok := params["start_date"]; ok {
		startDate = fmt.Sprint(s)
	}
	if e, ok := params["end_date"]; ok {
		endDate = fmt.Sprint(e)
	}

	switch source {
	case "csv":
		bars, err := loadCSV(path)
		if err != nil {
			return nil, fmt.Errorf("data_loader: %w", err)
		}
		return barsResult(bars), nil

	case "mock":
		bars := generateMockData(symbol, 252)
		return barsResult(bars), nil

	default:
		if nctx == nil || nctx.MarketReg == nil {
			return nil, fmt.Errorf("data_loader: market data registry not available (wired at app startup)")
		}
		reg := nctx.MarketReg
		if reg == nil {
			return nil, fmt.Errorf("data_loader: invalid market registry type")
		}

		now := time.Now()
		startUnix := parseDate(startDate, now.AddDate(-1, 0, 0)).Unix()
		endUnix := parseDate(endDate, now).Unix()

		mkt := market.MarketForSymbol(symbol)
		bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, interval, "", startUnix, endUnix)
		if err != nil {
			return nil, fmt.Errorf("data_loader: %w", err)
		}
		return barsResult(bars), nil
	}
}

func extractSeries(bars []market.OHLCVBar, sel func(market.OHLCVBar) float64) []float64 {
	s := make([]float64, len(bars))
	for i, b := range bars {
		s[i] = sel(b)
	}
	return s
}

func barsResult(bars []market.OHLCVBar) map[string]any {
	return map[string]any{
		"ohlcv":  bars,
		"close":  extractSeries(bars, func(b market.OHLCVBar) float64 { return b.Close }),
		"open":   extractSeries(bars, func(b market.OHLCVBar) float64 { return b.Open }),
		"high":   extractSeries(bars, func(b market.OHLCVBar) float64 { return b.High }),
		"low":    extractSeries(bars, func(b market.OHLCVBar) float64 { return b.Low }),
		"volume": extractSeries(bars, func(b market.OHLCVBar) float64 { return b.Volume }),
	}
}

func (n *DataLoaderNode) Validate() error {
	source := "auto"
	if s, ok := n.params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if source == "csv" {
		path := ""
		if p, ok := n.params["path"]; ok {
			path = fmt.Sprint(p)
		}
		if path == "" {
			return fmt.Errorf("data_loader: path is required for csv source")
		}
	}
	return nil
}

func loadCSV(path string) ([]market.OHLCVBar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[col] = i
	}

	var bars []market.OHLCVBar
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row: %w", err)
		}
		bars = append(bars, market.OHLCVBar{
			Symbol: "CSV",
			Date:   record[colIdx["date"]],
			Open:   parseFloat(record[colIdx["open"]]),
			High:   parseFloat(record[colIdx["high"]]),
			Low:    parseFloat(record[colIdx["low"]]),
			Close:  parseFloat(record[colIdx["close"]]),
			Volume: parseFloat(record[colIdx["volume"]]),
		})
	}
	return bars, nil
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func generateMockData(symbol string, n int) []market.OHLCVBar {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // mock 行情数据生成，非安全用途
	bars := make([]market.OHLCVBar, n)
	startDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	price := 50.0

	for i := 0; i < n; i++ {
		date := startDate.AddDate(0, 0, i)
		for date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			date = date.AddDate(0, 0, 1)
		}

		ret := (rng.NormFloat64()*0.02 + 0.001) * price
		open := price
		close := math.Max(open+ret, 0.01)
		high := math.Max(open, close) * (1 + math.Abs(rng.NormFloat64())*0.01)
		low := math.Min(open, close) * (1 - math.Abs(rng.NormFloat64())*0.01)
		volume := 1e6 + rng.Float64()*5e6
		price = close

		bars[i] = market.OHLCVBar{
			Symbol: symbol,
			Date:   date.Format("2006-01-02"),
			Open:   math.Round(open*100) / 100,
			High:   math.Round(high*100) / 100,
			Low:    math.Round(low*100) / 100,
			Close:  math.Round(close*100) / 100,
			Volume: math.Round(volume),
		}
	}
	return bars
}
