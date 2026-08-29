package nodes

import (
	"context"
	"encoding/json"
	"log/slog"
	"quantflow/internal/research"
	"quantflow/internal/workflow"
	"time"
)

// PredictionMarketNode fetches prediction market data and extracts
// probability-based trading signals. Degrades to mock data when the
// prediction market service is not configured.
type PredictionMarketNode struct {
	id     string
	params map[string]any
}

// NewPredictionMarketNode creates a new prediction market workflow node.
func NewPredictionMarketNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PredictionMarketNode{id: id, params: params}, nil
}

func (n *PredictionMarketNode) ID() string       { return n.id }
func (n *PredictionMarketNode) NodeType() string { return "prediction_market" }
func (n *PredictionMarketNode) Category() string { return "alternative_data" }

func (n *PredictionMarketNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "category", Type: workflow.PortString, Required: false},
		{Name: "min_prob_change", Type: workflow.PortNumber, Required: false},
	}
}

func (n *PredictionMarketNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "top_events", Type: workflow.PortSeries, Required: false},
		{Name: "signal", Type: workflow.PortSignal, Required: false},
		{Name: "signal_summary", Type: workflow.PortString, Required: false},
	}
}

func (n *PredictionMarketNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "category", Type: "string", Default: "", Description: "Event category filter"},
		{Name: "min_prob_change", Type: "number", Default: 0.05, Description: "Min probability change for signal"},
		{Name: "limit", Type: "number", Default: 20, Description: "Max events to fetch"},
	}
}

func (n *PredictionMarketNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var predictionMarketService workflow.PredictionMarketServiceInterface
	if nctx != nil {
		predictionMarketService = nctx.PredictionMarketService
	}
	category := resolveStringParam(params, n.params, "category", "")
	if v, ok := inputs["category"].(string); ok && v != "" {
		category = v
	}

	minProbChange := 0.05
	if v := resolveFloatParam(params, n.params, "min_prob_change"); v != 0 {
		minProbChange = v
	}
	if v, ok := inputs["min_prob_change"].(float64); ok && v != 0 {
		minProbChange = v
	}

	limit := int(resolveFloatParam(params, n.params, "limit"))
	if limit <= 0 {
		limit = 20
	}

	var output *research.SignalOutput
	var err error

	if predictionMarketService != nil {
		output, err = predictionMarketService.ExtractSignals(ctx, category, minProbChange)
	} else {
		slog.Warn("prediction market service not set, using mock")
		// Use the service's own mock data via a nil-adapter instance.
		mockSvc := research.NewPredictionMarketService(nil)
		events, _ := mockSvc.GetEvents(ctx, category, limit)
		output = &research.SignalOutput{
			Category:    category,
			Events:      events,
			Signal:      research.SignalSummary{Action: "hold", Confidence: 0.0, Description: "mock prediction signal"},
			GeneratedAt: time.Now().UTC(),
		}
	}
	if err != nil {
		slog.Warn("prediction market signal extraction failed", "error", err)
		mockSvc := research.NewPredictionMarketService(nil)
		events, _ := mockSvc.GetEvents(ctx, category, limit)
		output = &research.SignalOutput{
			Category:    category,
			Events:      events,
			Signal:      research.SignalSummary{Action: "hold", Confidence: 0.0, Description: "error fallback signal"},
			GeneratedAt: time.Now().UTC(),
		}
	}

	// Truncate events to limit
	events := output.Events
	if len(events) > limit {
		events = events[:limit]
	}
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		slog.Warn("prediction_market: marshal events", "error", err)
		eventsJSON = []byte("[]")
	}

	return map[string]any{
		"top_events":     string(eventsJSON),
		"signal":         signalToMap(output.Signal),
		"signal_summary": output.Signal.Description,
	}, nil
}

func (n *PredictionMarketNode) Validate() error { return nil }

// ── Helpers ───────────────────────────────────────────────────────

func signalToMap(s research.SignalSummary) map[string]any {
	return map[string]any{
		"action":      s.Action,
		"confidence":  s.Confidence,
		"description": s.Description,
	}
}
