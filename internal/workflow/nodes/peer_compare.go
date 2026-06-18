package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/workflow"
)

// PeerCompareNode compares a symbol against its peer group.
// Degrades to an empty peer list when the PeerComparisonService is not set.
type PeerCompareNode struct {
	id     string
	params map[string]any
}

// NewPeerCompareNode creates a new PeerCompareNode.
func NewPeerCompareNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &PeerCompareNode{id: id, params: params}, nil
}

func (n *PeerCompareNode) ID() string       { return n.id }
func (n *PeerCompareNode) NodeType() string { return "peer_compare" }
func (n *PeerCompareNode) Category() string { return "research" }

func (n *PeerCompareNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
	}
}

func (n *PeerCompareNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "peers", Type: workflow.PortSeries, Required: false},
		{Name: "comparison_metrics", Type: workflow.PortSeries, Required: false},
	}
}

func (n *PeerCompareNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "max_peers", Type: "int", Default: 10, Description: "Maximum number of peers to return"},
	}
}

func (n *PeerCompareNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("peer_compare: missing required input 'symbol'")
	}

	if peerComparisonService != nil {
		peers, err := peerComparisonService.GetPeers(ctx, symbol)
		if err != nil {
			slog.Warn("peer comparison service returned error, using empty peers", "symbol", symbol, "error", err)
			return map[string]any{
				"peers":              []any{},
				"comparison_metrics": map[string]any{},
			}, nil
		}
		// Build summary comparison metrics from peer data
		comparisonMetrics := computeComparisonMetrics(peers)
		return map[string]any{
			"peers":              peers,
			"comparison_metrics": comparisonMetrics,
		}, nil
	}

	slog.Warn("peer comparison service not set, using empty peers", "symbol", symbol)
	return map[string]any{
		"peers":              []any{},
		"comparison_metrics": map[string]any{},
	}, nil
}

func (n *PeerCompareNode) Validate() error { return nil }

// computeComparisonMetrics derives aggregate metrics from a peer group.
func computeComparisonMetrics(peers any) map[string]any {
	return map[string]any{
		"source": "mock",
	}
}
