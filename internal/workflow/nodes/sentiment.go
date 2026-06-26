package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// SentimentNode analyzes market sentiment for a symbol and outputs a trading signal.
// Degrades to neutral/mock data when no SentimentEngine is set.
type SentimentNode struct {
	id     string
	params map[string]any
}

// NewSentimentNode creates a new SentimentNode.
func NewSentimentNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SentimentNode{id: id, params: params}, nil
}

func (n *SentimentNode) ID() string       { return n.id }
func (n *SentimentNode) NodeType() string { return "sentiment" }
func (n *SentimentNode) Category() string { return "research" }

func (n *SentimentNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
		{Name: "news_text", Type: workflow.PortString, Required: false},
	}
}

func (n *SentimentNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "sentiment_score", Type: workflow.PortNumber, Required: false},
		{Name: "sentiment_label", Type: workflow.PortString, Required: false},
		{Name: "signal", Type: workflow.PortSignal, Required: false},
		{Name: "keywords", Type: workflow.PortSeries, Required: false},
	}
}

func (n *SentimentNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "text_type", Type: "string", Default: "news", Description: "Source type: news, social, filing"},
		{Name: "language", Type: "string", Default: "en", Description: "Text language: en, zh"},
	}
}

func (n *SentimentNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("sentiment: missing required input 'symbol'")
	}

	textContent := ""
	if t, ok := inputs["news_text"].(string); ok {
		textContent = t
	}

	textType := resolveStringParam(params, n.params, "text_type", "news")
	language := resolveStringParam(params, n.params, "language", "en")

	var output *research.SentimentOutput
	var err error

	if sentimentEngine != nil {
		output, err = sentimentEngine.AnalyzeSentiment(ctx, symbol, textContent, textType, language)
	} else {
		slog.Warn("sentiment engine not set, using mock")
		output = mockSentimentResult(symbol, textType)
	}
	if err != nil {
		output = mockSentimentResult(symbol, textType)
	}

	signal := sentimentToSignal(output.Score, output.Confidence)

	return map[string]any{
		"sentiment_score": output.Score,
		"sentiment_label": output.Label,
		"signal":          signal,
		"keywords":        output.Keywords,
	}, nil
}

func (n *SentimentNode) Validate() error { return nil }

// sentimentToSignal converts sentiment score to a trading signal.
// Uses the same ±0.15 threshold as the NLP pipeline for consistency.
func sentimentToSignal(score, confidence float64) map[string]any {
	action := "hold"
	if confidence > 0.3 {
		if score > 0.15 {
			action = "buy"
		} else if score < -0.15 {
			action = "sell"
		}
	}
	return map[string]any{
		"action":     action,
		"confidence": confidence,
	}
}

func mockSentimentResult(symbol, textType string) *research.SentimentOutput {
	return &research.SentimentOutput{
		Symbol:     symbol,
		Score:      0.0,
		Label:      "neutral",
		Confidence: 0.0,
		Keywords:   []string{"mock_data"},
		Source:     textType,
	}
}

// resolveStringParam resolves a string param, preferring runtime params over constructor params.
func resolveStringParam(runtime, constructor map[string]any, key, defaultVal string) string {
	if v, ok := runtime[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if v, ok := constructor[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
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
