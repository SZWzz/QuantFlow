package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// AllocationNode computes portfolio allocation breakdowns by market and sector.
type AllocationNode struct {
	id     string
	params map[string]any
}

// NewAllocationNode creates a new AllocationNode.
func NewAllocationNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &AllocationNode{id: id, params: params}, nil
}

func (n *AllocationNode) ID() string        { return n.id }
func (n *AllocationNode) NodeType() string  { return "allocation" }
func (n *AllocationNode) Category() string  { return "portfolio" }

func (n *AllocationNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "positions", Type: workflow.PortSeries, Required: false},
	}
}

func (n *AllocationNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "by_market", Type: workflow.PortSeries, Required: false},
		{Name: "by_sector", Type: workflow.PortSeries, Required: false},
	}
}

func (n *AllocationNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *AllocationNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	// TODO: Compute real allocation from positions input.
	_ = inputs["positions"]
	return nil, fmt.Errorf("allocation: not yet implemented — returns placeholder data has been removed; real computation requires position-to-market/sector mapping")
}

func (n *AllocationNode) Validate() error { return nil }
