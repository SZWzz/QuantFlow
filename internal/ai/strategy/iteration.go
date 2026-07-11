package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// IterationRequest contains the backtest results for strategy optimization.
type IterationRequest struct {
	WorkflowJSON string              `json:"workflow_json"` // original workflow as JSON
	Metrics      *IterationMetrics   `json:"metrics"`
	Goal         string              `json:"goal"` // "max_sharpe" | "min_drawdown" | "max_return"
}

// IterationMetrics summarizes backtest performance.
type IterationMetrics struct {
	SharpeRatio  float64 `json:"sharpe_ratio"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	TotalReturn  float64 `json:"total_return"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
}

// IterationResult contains the parameter tuning suggestions.
type IterationResult struct {
	Analysis string         `json:"analysis"` // natural language analysis
	Changes  []ParamChange  `json:"changes"`  // suggested parameter changes
}

// ParamChange describes a single parameter adjustment.
type ParamChange struct {
	NodeID   string  `json:"node_id"`
	Param    string  `json:"param"`
	From     float64 `json:"from"`
	To       float64 `json:"to"`
	Reason   string  `json:"reason"`
}

// IterationAgent analyzes backtest results and suggests parameter improvements.
type IterationAgent struct {
	llm      LLMCaller
	maxRounds int
}

// NewIterationAgent creates a strategy iteration agent.
func NewIterationAgent(llm LLMCaller) *IterationAgent {
	return &IterationAgent{llm: llm, maxRounds: 5}
}

// Analyze backtest results and suggest parameter changes.
func (a *IterationAgent) Analyze(ctx context.Context, req IterationRequest) (*IterationResult, error) {
	systemPrompt := `You are a quantitative trading strategy optimizer. Analyze the backtest results and suggest parameter changes.

## Output Format (JSON only)
{
  "analysis": "one sentence describing what's wrong and how to fix it",
  "changes": [
    {"node_id": "node_1", "param": "period", "from": 20, "to": 10, "reason": "shorter period may capture trends sooner"}
  ]
}

## Guidelines
- If Sharpe < 0.5: tighten stop-loss, reduce holding period
- If MaxDrawdown > 20%: reduce position size, add stop-loss
- If WinRate < 40%: adjust entry signal thresholds
- If ProfitFactor < 1.2: filter out noise, add confirmation signals
- Small changes (10-20%) are safer than large jumps`

	metricsJSON, _ := json.MarshalIndent(req.Metrics, "", "  ")
	userPrompt := fmt.Sprintf("Goal: %s\nMetrics:\n%s\nWorkflow:\n%s",
		req.Goal, string(metricsJSON), req.WorkflowJSON)

	response, err := a.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("iteration agent: %w", err)
	}

	jsonStr := extractJSON(response)
	var result IterationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("iteration agent: parse error: %w (raw: %s)", err, truncate(jsonStr, 200))
	}

	return &result, nil
}

// Optimize runs multiple rounds of analyze→modify→backtest until convergence or maxRounds.
// The caller is responsible for applying changes and running backtests between rounds.
func (a *IterationAgent) Optimize(ctx context.Context, initial IterationRequest) ([]IterationResult, error) {
	var history []IterationResult
	current := initial

	for round := 1; round <= a.maxRounds; round++ {
		result, err := a.Analyze(ctx, current)
		if err != nil {
			return history, fmt.Errorf("round %d: %w", round, err)
		}
		history = append(history, *result)

		// No more changes suggested = converged
		if len(result.Changes) == 0 {
			break
		}

		// The caller would apply changes and re-run backtest,
		// then call Analyze again with updated metrics.
		// For now, we just collect the results.
	}

	return history, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func containsAny(s, substr string) bool {
	return strings.Contains(s, substr)
}
