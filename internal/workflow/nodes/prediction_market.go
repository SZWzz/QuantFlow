package nodes

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
	"quantflow/internal/workflow"
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
func (n *PredictionMarketNode) Category() string  { return "alternative_data" }

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

func (n *PredictionMarketNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
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

	var output *research.SignalOutput
	var err error

	if predictionMarketService != nil {
		output, err = predictionMarketService.ExtractSignals(ctx, category, minProbChange)
	} else {
		slog.Warn("prediction market service not set, using mock")
		output = mockPredictionSignal(category)
	}
	if err != nil {
		slog.Warn("prediction market signal extraction failed", "error", err)
		output = mockPredictionSignal(category)
	}

	eventsJSON, _ := json.Marshal(output.Events)

	return map[string]any{
		"top_events":     string(eventsJSON),
		"signal":         signalToMap(output.Signal),
		"signal_summary": output.Signal.Description,
	}, nil
}

func (n *PredictionMarketNode) Validate() error { return nil }

// ── Helpers ───────────────────────────────────────────────────────

func mockPredictionSignal(category string) *research.SignalOutput {
	return &research.SignalOutput{
		Category:    category,
		Events:      mockPredictionEventsForNode(category),
		Signal:      research.SignalSummary{Action: "hold", Confidence: 0.0, Description: "mock prediction signal"},
		GeneratedAt: time.Now().UTC(),
	}
}

func signalToMap(s research.SignalSummary) map[string]any {
	return map[string]any{
		"action":      s.Action,
		"confidence":  s.Confidence,
		"description": s.Description,
	}
}

// mockPredictionEventsForNode provides node-level mock data (minimal, no
// dependency on PredictionMarketService).
func mockPredictionEventsForNode(category string) []adapters.PredictionEvent {
	all := []adapters.PredictionEvent{
		{
			ID: "fed-rate-cut-july", Title: "Fed cuts rates by July 2026?",
			Category: "economics", Volume: 2_500_000, Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes", Label: "Yes", Price: 0.35, Change24h: 0.03},
				{ID: "no", Label: "No", Price: 0.65, Change24h: -0.03},
			},
		},
		{
			ID: "bitcoin-100k", Title: "Bitcoin breaks $100K by Q3 2026?",
			Category: "crypto", Volume: 4_200_000, Status: "open",
			Outcomes: []adapters.PredictionOutcome{
				{ID: "yes", Label: "Yes", Price: 0.28, Change24h: -0.05},
				{ID: "no", Label: "No", Price: 0.72, Change24h: 0.05},
			},
		},
	}
	if category != "" {
		var filtered []adapters.PredictionEvent
		for _, e := range all {
			if e.Category == category {
				filtered = append(filtered, e)
			}
		}
		return filtered
	}
	return all
}

// resolveFloatParam resolves a float64 param, preferring runtime params over constructor params.
func resolveFloatParam(runtime, constructor map[string]any, key string) float64 {
	if v, ok := runtime[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	if v, ok := constructor[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
