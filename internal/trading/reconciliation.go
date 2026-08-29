package trading

import (
	"context"
	"encoding/json"
	"time"
)

// ReconciliationReport compares OMS positions against broker positions.
type ReconciliationReport struct {
	ID         int64                `json:"id"`
	CreatedAt  time.Time            `json:"created_at"`
	BrokerName string               `json:"broker_name"`
	MatchCount int                  `json:"match_count"`
	DiffCount  int                  `json:"diff_count"`
	Dirt       string               `json:"dirt"` // "clean" or "dirty"
	Diffs      []ReconciliationDiff `json:"diffs"`
	OMSOnly    []string             `json:"oms_only"`    // symbols in OMS but not broker
	BrokerOnly []string             `json:"broker_only"` // symbols in broker but not OMS
}

// ReconciliationDiff represents a single position mismatch.
type ReconciliationDiff struct {
	Symbol         string  `json:"symbol"`
	OMSQuantity    float64 `json:"oms_quantity"`
	BrokerQty      float64 `json:"broker_quantity"`
	OMSAvgPrice    float64 `json:"oms_avg_price"`
	BrokerAvgPrice float64 `json:"broker_avg_price"`
}

// ReconcileAll compares OMS positions against broker positions for all configured brokers.
func ReconcileAll(oms *OMS, brokers map[string]Broker) []*ReconciliationReport {
	var reports []*ReconciliationReport

	for name, broker := range brokers {
		if !broker.IsConnected() {
			continue
		}
		report := &ReconciliationReport{
			CreatedAt:  time.Now(),
			BrokerName: name,
			Dirt:       "clean",
		}

		// Collect OMS positions
		omsPositions := oms.GetAllPositions()
		omsMap := make(map[string]*Position)
		for _, p := range omsPositions {
			omsMap[p.Symbol] = p
		}

		// Collect broker positions
		brokerPositions, err := broker.GetPositions(context.Background())
		if err != nil {
			report.Dirt = "dirty"
			reports = append(reports, report)
			continue
		}

		brokerMap := make(map[string]*Position)
		for _, p := range brokerPositions {
			brokerMap[p.Symbol] = p
		}

		// Match positions
		for symbol, omsPos := range omsMap {
			if brokerPos, ok := brokerMap[symbol]; ok {
				if omsPos.Quantity == brokerPos.Quantity {
					report.MatchCount++
				} else {
					report.DiffCount++
					report.Diffs = append(report.Diffs, ReconciliationDiff{
						Symbol:         symbol,
						OMSQuantity:    omsPos.Quantity,
						BrokerQty:      brokerPos.Quantity,
						OMSAvgPrice:    omsPos.AvgPrice,
						BrokerAvgPrice: brokerPos.AvgPrice,
					})
				}
			} else {
				report.DiffCount++
				report.OMSOnly = append(report.OMSOnly, symbol)
			}
		}
		for symbol := range brokerMap {
			if _, ok := omsMap[symbol]; !ok {
				report.DiffCount++
				report.BrokerOnly = append(report.BrokerOnly, symbol)
			}
		}

		if report.DiffCount > 0 {
			report.Dirt = "dirty"
		}
		reports = append(reports, report)
	}

	return reports
}

// EncodeReconciliationReport marshals to JSON.
func EncodeReconciliationReport(r *ReconciliationReport) ([]byte, error) {
	return json.Marshal(r)
}

// DecodeReconciliationReport unmarshals from JSON.
func DecodeReconciliationReport(data []byte) (*ReconciliationReport, error) {
	var r ReconciliationReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
