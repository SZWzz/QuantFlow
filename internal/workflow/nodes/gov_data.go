package nodes

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// GovDataNode fetches economic indicator data from FRED and extracts
// macro-economic trading signals. Degrades to mock data when the
// govdata service is not configured.
type GovDataNode struct {
	id     string
	params map[string]any
}

// NewGovDataNode creates a new gov_data workflow node.
func NewGovDataNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &GovDataNode{id: id, params: params}, nil
}

func (n *GovDataNode) ID() string       { return n.id }
func (n *GovDataNode) NodeType() string  { return "gov_data" }
func (n *GovDataNode) Category() string  { return "alternative_data" }

func (n *GovDataNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "indicator", Type: workflow.PortString, Required: false},
		{Name: "country", Type: workflow.PortString, Required: false},
	}
}

func (n *GovDataNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "macro_signal", Type: workflow.PortSignal, Required: false},
		{Name: "latest_value", Type: workflow.PortNumber, Required: false},
		{Name: "change", Type: workflow.PortNumber, Required: false},
	}
}

func (n *GovDataNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "indicator", Type: "string", Default: "all", Description: "FRED series ID or 'all' for all indicators"},
		{Name: "lookback", Type: "number", Default: 12, Description: "Number of historical data points"},
	}
}

func (n *GovDataNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var govDataService *research.GovDataService
	if nctx != nil {
		govDataService, _ = nctx.GovDataService.(*research.GovDataService)
	}
	indicator := resolveStringParam(params, n.params, "indicator", "all")
	if v, ok := inputs["indicator"].(string); ok && v != "" {
		indicator = v
	}

	lookback := int(resolveFloatParam(params, n.params, "lookback"))
	if lookback <= 0 {
		lookback = 12
	}

	var signals []research.MacroSignal
	var targetSignal *research.MacroSignal
	var err error

	if govDataService != nil {
		signals, err = govDataService.GetAllSignals(ctx)
	} else {
		slog.Warn("govdata service not set, using mock")
		mockSvc := research.NewGovDataService(nil)
		signals, _ = mockSvc.GetAllSignals(ctx)
	}
	if err != nil {
		slog.Warn("govdata signal extraction failed", "error", err)
		mockSvc := research.NewGovDataService(nil)
		signals, _ = mockSvc.GetAllSignals(ctx)
	}

	// Find the matching indicator or use aggregate signal
	if indicator != "" && indicator != "all" {
		for i := range signals {
			if signals[i].IndicatorID == indicator {
				targetSignal = &signals[i]
				break
			}
		}
	}

	// Aggregate: count bullish vs bearish signals
	bullishCount := 0
	bearishCount := 0
	for _, s := range signals {
		switch s.Signal {
		case "bullish":
			bullishCount++
		case "bearish":
			bearishCount++
		}
	}

	// Build macro signal output
	action := "hold"
	confidence := 0.0
	description := ""

	if targetSignal != nil {
		switch targetSignal.Signal {
		case "bullish":
			action = "buy"
			confidence = 0.5
		case "bearish":
			action = "sell"
			confidence = 0.5
		default:
			action = "hold"
			confidence = 0.1
		}
		description = targetSignal.NameCN + ": " + targetSignal.Direction + " " + targetSignal.Signal
	} else {
		// Aggregate signal
		total := len(signals)
		if bullishCount > total/3 && bullishCount > bearishCount {
			action = "buy"
			confidence = float64(bullishCount) / float64(total)
			description = "宏观指标总体偏多"
		} else if bearishCount > total/3 && bearishCount > bullishCount {
			action = "sell"
			confidence = float64(bearishCount) / float64(total)
			description = "宏观指标总体偏空"
		} else {
			action = "hold"
			confidence = 0.2
			description = "宏观指标信号中性/混合"
		}
	}

	signalsJSON, err := json.Marshal(signals)
	if err != nil {
		slog.Warn("gov_data: marshal signals", "error", err)
		signalsJSON = []byte("[]")
	}

	return map[string]any{
		"macro_signal": map[string]any{
			"action":      action,
			"confidence":  confidence,
			"description": description,
			"all_signals": signalsJSON,
		},
		"latest_value": func() float64 {
			if targetSignal != nil {
				return targetSignal.LatestValue
			}
			return float64(bullishCount) / float64(max(len(signals), 1))
		}(),
		"change": func() float64 {
			if targetSignal != nil {
				return targetSignal.Change
			}
			return float64(bullishCount-bearishCount) / float64(max(len(signals), 1)) * 100
		}(),
	}, nil
}

func (n *GovDataNode) Validate() error { return nil }

// blank assignment to keep unused import
var _ = time.Now
