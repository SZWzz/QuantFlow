package market

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// MacroIndicator is a single macroeconomic data point.
type MacroIndicator struct {
	Country string  `json:"country"`
	Name    string  `json:"name"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	Date    string  `json:"date"`
	Change  float64 `json:"change"`
}

// MacroSnapshot groups macro indicators by category.
type MacroSnapshot struct {
	Growth    []MacroIndicator `json:"growth"`
	Inflation []MacroIndicator `json:"inflation"`
	Monetary  []MacroIndicator `json:"monetary"`
	Policy    []MacroIndicator `json:"policy"`
	UpdatedAt string           `json:"updated_at"`
}

// MacroFetcher fetches raw macro data (injected for CN/US data sources).
type MacroFetcher interface {
	FetchMacroData(dataType string) ([]byte, error)
}

// MacroService routes macro data requests to configured fetchers.
type MacroService struct {
	fetcher MacroFetcher
	db      *sql.DB
}

// NewMacroService creates a new MacroService.
func NewMacroService(fetcher MacroFetcher, db *sql.DB) *MacroService {
	return &MacroService{
		fetcher: fetcher,
		db:      db,
	}
}

// GetMacroSnapshot returns the latest macro snapshot for a country.
func (s *MacroService) GetMacroSnapshot(country string) (*MacroSnapshot, error) {
	if s.fetcher == nil {
		return emptySnapshot(), nil
	}

	raw, err := s.fetcher.FetchMacroData("macro_cn_summary")
	if err != nil {
		return emptySnapshot(), nil
	}

	var flat []struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
		Date  string  `json:"date"`
	}
	if err := json.Unmarshal(raw, &flat); err != nil {
		var grouped map[string][]struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
			Date  string  `json:"date"`
		}
		if err2 := json.Unmarshal(raw, &grouped); err2 != nil {
			return emptySnapshot(), fmt.Errorf("parse macro: %w", err)
		}
		var all []struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
			Date  string  `json:"date"`
		}
		for _, items := range grouped {
			all = append(all, items...)
		}
		return classifyMacroIndicators(all), nil
	}
	return classifyMacroIndicators(flat), nil
}

func emptySnapshot() *MacroSnapshot {
	return &MacroSnapshot{
		Growth:    []MacroIndicator{},
		Inflation: []MacroIndicator{},
		Monetary:  []MacroIndicator{},
		Policy:    []MacroIndicator{},
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
}

func classifyMacroIndicators(items []struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Date  string  `json:"date"`
},
) *MacroSnapshot {
	s := emptySnapshot()
	for _, item := range items {
		mi := MacroIndicator{
			Country: "CN",
			Name:    item.Name,
			Value:   item.Value,
			Unit:    item.Unit,
			Date:    item.Date,
		}
		switch {
		case containsAny(item.Name, "GDP", "PMI", "工业", "零售", "固投", "地产", "失业"):
			s.Growth = append(s.Growth, mi)
		case containsAny(item.Name, "CPI", "PPI", "通胀", "价格"):
			s.Inflation = append(s.Inflation, mi)
		case containsAny(item.Name, "M2", "社融", "信贷", "SHIBOR", "LPR", "准备金", "外汇", "货币"):
			s.Monetary = append(s.Monetary, mi)
		default:
			s.Policy = append(s.Policy, mi)
		}
	}
	return s
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if len(s) >= len(kw) {
			for i := 0; i <= len(s)-len(kw); i++ {
				if s[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}
