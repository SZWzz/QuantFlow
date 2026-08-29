package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

type MACDNode struct {
	id     string
	params map[string]any
}

func NewMACDNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &MACDNode{id: id, params: params}, nil
}

func (n *MACDNode) ID() string       { return n.id }
func (n *MACDNode) NodeType() string { return "macd" }
func (n *MACDNode) Category() string { return "indicator" }

func (n *MACDNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "prices", Type: workflow.PortSeries, Required: true}}
}

func (n *MACDNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "macd_line", Type: workflow.PortSeries, Required: false},
		{Name: "signal_line", Type: workflow.PortSeries, Required: false},
		{Name: "histogram", Type: workflow.PortSeries, Required: false},
	}
}

func (n *MACDNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "fast", Type: "number", Default: "12", Description: "Fast EMA period"},
		{Name: "slow", Type: "number", Default: "26", Description: "Slow EMA period"},
		{Name: "signal", Type: "number", Default: "9", Description: "Signal line period"},
	}
}

func (n *MACDNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	prices := extractFloatSlice(inputs["prices"])
	if prices == nil {
		return nil, fmt.Errorf("macd: prices input required")
	}

	fast := int(getFloatParam(params, "fast", 12))
	slow := int(getFloatParam(params, "slow", 26))
	sig := int(getFloatParam(params, "signal", 9))

	emaFast := ema(prices, fast)
	emaSlow := ema(prices, slow)

	dataLen := len(prices)
	macdLine := make([]float64, dataLen)
	histogram := make([]float64, dataLen)
	for i := 0; i < dataLen; i++ {
		macdLine[i] = emaFast[i] - emaSlow[i]
	}
	signalLine := ema(macdLine, sig)
	for i := 0; i < dataLen; i++ {
		histogram[i] = macdLine[i] - signalLine[i]
	}

	return map[string]any{
		"macd_line": macdLine, "signal_line": signalLine, "histogram": histogram,
	}, nil
}

func (n *MACDNode) Validate() error { return nil }

func ema(data []float64, period int) []float64 {
	result := make([]float64, len(data))
	if len(data) == 0 || period <= 0 {
		return result
	}
	k := 2.0 / float64(period+1)
	result[0] = data[0]
	for i := 1; i < len(data); i++ {
		result[i] = data[i]*k + result[i-1]*(1-k)
	}
	return result
}

// extractFloatSlice is defined in utils.go
