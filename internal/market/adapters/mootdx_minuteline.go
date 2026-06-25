// mootdx_minuteline.go — mootdx adapter for intraday minute-line data.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	pb "quantflow/internal/python/proto"
	"quantflow/internal/market"
)

// FetchMinuteLine returns today's minute-by-minute price/volume ticks
// via the mootdx Python sidecar.
func (a *MootdxAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("mootdx: Python sidecar not connected")
	}

	resp, err := a.dataClient.FetchData(context.Background(), &pb.FetchDataRequest{
		Source:   "mootdx",
		DataType: "quote",
		Symbols:  []string{symbol},
		Params:   map[string]string{"field": "minute"},
	})
	if err != nil {
		return nil, fmt.Errorf("mootdx minuteline: %w", err)
	}

	var raw []rawMinuteTick
	if err := json.Unmarshal(resp.Data, &raw); err != nil {
		return nil, fmt.Errorf("mootdx minuteline parse: %w", err)
	}

	ticks := make([]market.MinuteTick, 0, len(raw))
	for _, r := range raw {
		ticks = append(ticks, market.MinuteTick{
			Time:     r.Time,
			Price:    r.Price,
			Volume:   r.Volume,
			AvgPrice: r.AvgPrice,
		})
	}
	return ticks, nil
}

// rawMinuteTick mirrors the JSON from Python sidecar's _fetch_mootdx_quote output.
type rawMinuteTick struct {
	Time     string  `json:"time"`
	Price    float64 `json:"price"`
	Volume   float64 `json:"volume"`
	AvgPrice float64 `json:"avg_price"`
}
