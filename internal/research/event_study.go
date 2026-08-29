package research

import (
	"context"
	"fmt"
	"math"
	"quantflow/internal/market"
	"time"
)

// EventStudyResult holds CAR analysis around an event date.
type EventStudyResult struct {
	Symbol    string    `json:"symbol"`
	EventDate string    `json:"event_date"`
	EventType string    `json:"event_type"`
	Window    int       `json:"window"`
	CAR       float64   `json:"car"`
	DailyAR   []DailyAR `json:"daily_ar"`
}

// DailyAR is a single day's abnormal return.
type DailyAR struct {
	Date   string  `json:"date"`
	Day    int     `json:"day"`
	AR     float64 `json:"ar"`
	CAR    float64 `json:"car"`
	StockR float64 `json:"stock_r"`
	BenchR float64 `json:"bench_r"`
}

// EventStudyService computes CAR around corporate events.
type EventStudyService struct{}

// NewEventStudyService creates a new EventStudyService.
func NewEventStudyService() *EventStudyService {
	return &EventStudyService{}
}

// ComputeCAR calculates cumulative abnormal return around an event.
func (s *EventStudyService) ComputeCAR(
	ctx context.Context,
	stockBars, benchBars []market.OHLCVBar,
	eventDate string,
	window int,
) (*EventStudyResult, error) {
	if len(stockBars) == 0 {
		return nil, fmt.Errorf("no stock data")
	}

	eventTime, err := time.Parse("2006-01-02", eventDate)
	if err != nil {
		return nil, fmt.Errorf("parse event date: %w", err)
	}

	// Build date-indexed maps
	stockMap := barsToMap(stockBars)
	benchMap := barsToMap(benchBars)

	result := &EventStudyResult{
		Symbol:    stockBars[0].Symbol,
		EventDate: eventDate,
		Window:    window,
		DailyAR:   make([]DailyAR, 0, window*2+1),
	}

	var cumAR float64
	for d := -window; d <= window; d++ {
		date := eventTime.AddDate(0, 0, d).Format("2006-01-02")

		sb, sok := stockMap[date]
		bb, bok := benchMap[date]

		stockR := 0.0
		benchR := 0.0
		if sok && sb.Open > 0 {
			stockR = (sb.Close - sb.Open) / sb.Open
		}
		if bok && bb.Open > 0 {
			benchR = (bb.Close - bb.Open) / bb.Open
		}

		ar := stockR - benchR
		cumAR += ar

		result.DailyAR = append(result.DailyAR, DailyAR{
			Date:   date,
			Day:    d,
			AR:     math.Round(ar*10000) / 100,
			CAR:    math.Round(cumAR*10000) / 100,
			StockR: math.Round(stockR*10000) / 100,
			BenchR: math.Round(benchR*10000) / 100,
		})
	}

	result.CAR = math.Round(cumAR*10000) / 100
	return result, nil
}

func barsToMap(bars []market.OHLCVBar) map[string]market.OHLCVBar {
	m := make(map[string]market.OHLCVBar, len(bars))
	for _, b := range bars {
		m[b.Date] = b
	}
	return m
}
