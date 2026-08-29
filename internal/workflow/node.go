// Package workflow provides the workflow engine — DAG execution, node registry,
// and the BaseNode interface that all nodes implement.
package workflow

import "context"

// PortType defines the data type flowing through ports.
type PortType string

const (
	PortOHLCV   PortType = "ohlcv"
	PortSeries  PortType = "series" // []float64
	PortSignal  PortType = "signal" // buy/sell/hold + confidence
	PortString  PortType = "string"
	PortNumber  PortType = "number"  // float64
	PortBoolean PortType = "boolean" // bool
	PortAny     PortType = "any"
)

// PortDefinition describes an input or output port.
type PortDefinition struct {
	Name     string   `json:"name"`
	Type     PortType `json:"type"`
	Required bool     `json:"required"`
}

// ParamDef describes a configurable parameter for a node.
type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "int", "float", "string", "bool", "string_array"
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// BaseNode is the interface every workflow node must implement.
type BaseNode interface {
	ID() string
	NodeType() string
	Category() string
	InputPorts() []PortDefinition
	OutputPorts() []PortDefinition
	ParamSchema() []ParamDef
	Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *NodeContext) (map[string]any, error)
	Validate() error
}
