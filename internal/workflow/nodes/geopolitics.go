package nodes

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// GeopoliticsNode fetches geopolitical risk data and extracts
// coverage-and-tone-based risk signals. Degrades to mock data when the
// geopolitics service is not configured.
type GeopoliticsNode struct {
	id     string
	params map[string]any
}

// NewGeopoliticsNode creates a new geopolitics workflow node.
func NewGeopoliticsNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &GeopoliticsNode{id: id, params: params}, nil
}

func (n *GeopoliticsNode) ID() string       { return n.id }
func (n *GeopoliticsNode) NodeType() string  { return "geopolitics" }
func (n *GeopoliticsNode) Category() string  { return "alternative_data" }

func (n *GeopoliticsNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "topic", Type: workflow.PortString, Required: false},
		{Name: "region", Type: workflow.PortString, Required: false},
	}
}

func (n *GeopoliticsNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "risk_signal", Type: workflow.PortSignal, Required: false},
		{Name: "risk_score", Type: workflow.PortNumber, Required: false},
		{Name: "tone", Type: workflow.PortNumber, Required: false},
	}
}

func (n *GeopoliticsNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "topic", Type: "string", Default: "", Description: "Geopolitical topic ID (e.g., taiwan-strait, middle-east)"},
		{Name: "min_vol_change", Type: "number", Default: 50, Description: "Minimum volume change % for risk signal"},
	}
}

func (n *GeopoliticsNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var geopoliticsService workflow.GeopoliticsServiceInterface
	if nctx != nil {
		geopoliticsService = nctx.GeopoliticsService
	}
	topic := resolveStringParam(params, n.params, "topic", "")
	if v, ok := inputs["topic"].(string); ok && v != "" {
		topic = v
	}

	minVolChange := 50.0
	if v := resolveFloatParam(params, n.params, "min_vol_change"); v != 0 {
		minVolChange = v
	}

	var risks []research.TopicRisk
	var err error

	if geopoliticsService != nil {
		risks, err = geopoliticsService.ExtractRiskSignals(ctx, minVolChange)
	} else {
		slog.Warn("geopolitics service not set, using mock")
		mockSvc := research.NewGeopoliticsService(nil)
		risks, _ = mockSvc.ExtractRiskSignals(ctx, minVolChange)
	}
	if err != nil {
		slog.Warn("geopolitics risk signal extraction failed", "error", err)
		mockSvc := research.NewGeopoliticsService(nil)
		risks, _ = mockSvc.ExtractRiskSignals(ctx, minVolChange)
	}

	// Find the matching topic or use the first risk signal
	var matchedRisk *research.TopicRisk
	if topic != "" {
		for i := range risks {
			if risks[i].ID == topic {
				matchedRisk = &risks[i]
				break
			}
		}
	}
	if matchedRisk == nil && len(risks) > 0 {
		matchedRisk = &risks[0]
	}
	if matchedRisk == nil {
		// Fallback: create a default low-risk signal
		matchedRisk = &research.TopicRisk{
			ID:        topic,
			Title:     topic,
			RiskLevel: "low",
			Tone:      0,
			UpdatedAt: time.Now().UnixMilli(),
		}
	}

	// Build risk signal output
	action := "hold"
	confidence := 0.0
	switch matchedRisk.RiskLevel {
	case "high":
		action = "sell"
		confidence = 0.8
	case "medium":
		action = "sell"
		confidence = 0.4
	case "low":
		action = "hold"
		confidence = 0.1
	}

	description := fmt.Sprintf("%s: %s risk, tone=%.1f", matchedRisk.Title, matchedRisk.RiskLevel, matchedRisk.Tone)

	riskScore := 0.0
	switch matchedRisk.RiskLevel {
	case "high":
		riskScore = 0.8
	case "medium":
		riskScore = 0.4
	case "low":
		riskScore = 0.1
	}

	return map[string]any{
		"risk_signal": map[string]any{
			"action":      action,
			"confidence":  confidence,
			"description": description,
		},
		"risk_score": riskScore,
		"tone":       matchedRisk.Tone,
	}, nil
}

func (n *GeopoliticsNode) Validate() error { return nil }
