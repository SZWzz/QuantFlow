package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type WatchlistNode struct {
	id     string
	params map[string]any
}

func NewWatchlistNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &WatchlistNode{id: id, params: params}, nil
}

func (n *WatchlistNode) ID() string       { return n.id }
func (n *WatchlistNode) NodeType() string { return "watchlist" }
func (n *WatchlistNode) Category() string { return "data" }

func (n *WatchlistNode) InputPorts() []workflow.PortDefinition  { return nil }
func (n *WatchlistNode) OutputPorts() []workflow.PortDefinition { return []workflow.PortDefinition{{Name: "symbols", Type: workflow.PortSeries}} }
func (n *WatchlistNode) ParamSchema() []workflow.ParamDef       { return nil }
func (n *WatchlistNode) Validate() error                        { return nil }

func (n *WatchlistNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	return map[string]any{"symbols": []any{}}, nil
}
