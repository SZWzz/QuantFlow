package research

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pb "quantflow/internal/python/proto"
	"quantflow/internal/python"
)

// InsiderTradingService fetches insider transactions from SEC EDGAR Form 4
// filings via the Python sidecar (edgartools library).
type InsiderTradingService struct {
	bridge *python.PythonBridge
}

// NewInsiderTradingService creates a new InsiderTradingService.
// Pass a non-nil bridge to fetch real SEC data; nil bridge gracefully
// returns an empty slice.
func NewInsiderTradingService(bridge *python.PythonBridge) *InsiderTradingService {
	return &InsiderTradingService{bridge: bridge}
}

// pyInsiderRow matches the JSON shape returned by sec_insider.get_insider_transactions().
type pyInsiderRow struct {
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	Type       string  `json:"type"`
	Shares     float64 `json:"shares"`
	Price      float64 `json:"price"`
	Value      float64 `json:"value"`
	Date       string  `json:"date"`
	FilingDate string  `json:"filing_date"`
}

// GetInsiderTrades returns insider transactions for a symbol via SEC EDGAR.
// Returns nil, nil when the Python sidecar is unavailable (graceful degradation).
func (s *InsiderTradingService) GetInsiderTrades(ctx context.Context, symbol string) ([]InsiderTransaction, error) {
	if s.bridge == nil {
		slog.Debug("insider_trades: no bridge", "symbol", symbol)
		return nil, nil
	}

	client := python.NewDataClient(s.bridge)
	req := &pb.FetchDataRequest{
		Source:   "sec",
		DataType: "insider",
		Symbols:  []string{symbol},
	}
	resp, err := client.FetchData(ctx, req)
	if err != nil {
		slog.Warn("insider_trades: fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}

	var parsed struct {
		Data  []pyInsiderRow `json:"data"`
		Items []pyInsiderRow `json:"items"`
	}
	if err := json.Unmarshal(resp.Data, &parsed); err != nil {
		return nil, fmt.Errorf("insider_trades: parse: %w", err)
	}

	rows := parsed.Data
	if len(rows) == 0 {
		rows = parsed.Items
	}

	txns := make([]InsiderTransaction, 0, len(rows))
	for _, r := range rows {
		txns = append(txns, InsiderTransaction{
			Name:   r.Name,
			Role:   r.Role,
			Type:   normalizeInsiderType(r.Type),
			Shares: int64(r.Shares),
			Price:  r.Price,
			Value:  r.Value,
			Date:   firstNonEmptyStr(r.Date, r.FilingDate),
		})
	}

	slog.Debug("insider_trades: done", "symbol", symbol, "count", len(txns))
	return txns, nil
}

func normalizeInsiderType(t string) string {
	switch t {
	case "P-Purchase", "purchase", "buy", "Buy":
		return "buy"
	case "S-Sale", "sale", "sell", "Sell":
		return "sell"
	case "A-Award", "award", "grant":
		return "award"
	default:
		return t
	}
}

func firstNonEmptyStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
