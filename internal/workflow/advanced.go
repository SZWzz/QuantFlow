package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"
)

// ── Backtest Config & Result ──────────────────────────────────────────

// BacktestConfig defines walk-forward backtesting parameters.
type BacktestConfig struct {
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	TrainWindow int    `json:"train_window"` // days
	TestWindow  int    `json:"test_window"`
	StepSize    int    `json:"step_size"`
}

// BacktestWindowResult holds per-window performance.
type BacktestWindowResult struct {
	TrainStart string             `json:"train_start"`
	TrainEnd   string             `json:"train_end"`
	TestStart  string             `json:"test_start"`
	TestEnd    string             `json:"test_end"`
	Metrics    map[string]float64 `json:"metrics"`
	Error      string             `json:"error,omitempty"`
}

// BacktestResult aggregates all walk-forward windows.
type BacktestResult struct {
	Status     string                `json:"status"`
	Config     BacktestConfig        `json:"config"`
	Windows    []BacktestWindowResult `json:"windows"`
	AggMetrics map[string]float64    `json:"agg_metrics"`
	StartedAt  time.Time             `json:"started_at"`
	FinishedAt time.Time             `json:"finished_at"`
}

// ExecuteBacktest runs walk-forward backtesting.
func (e *Engine) ExecuteBacktest(ctx context.Context, wf *Workflow, cfg BacktestConfig) (*BacktestResult, error) {
	result := &BacktestResult{Status: "running", Config: cfg, StartedAt: time.Now()}
	defer func() { result.FinishedAt = time.Now() }()

	windows, err := buildWindows(cfg)
	if err != nil {
		result.Status = "failed"
		return result, err
	}

	var allReturns, sharpes []float64
	for i, w := range windows {
		slog.Info("backtest: window", "idx", i, "test", w.testStart+"→"+w.testEnd)
		e.nctx.BacktestStart = w.testStart
		e.nctx.BacktestEnd = w.testEnd

		winWf := wf.Clone()
		execResult, execErr := e.Execute(ctx, winWf)
		wr := BacktestWindowResult{
			TrainStart: w.trainStart, TrainEnd: w.trainEnd,
			TestStart: w.testStart, TestEnd: w.testEnd,
		}

		if execErr != nil {
			wr.Error = execErr.Error()
		} else {
			wr.Metrics = extractMetrics(execResult)
		}
		result.Windows = append(result.Windows, wr)

		if r, ok := wr.Metrics["total_return"]; ok {
			allReturns = append(allReturns, r)
		}
		if s, ok := wr.Metrics["sharpe"]; ok {
			sharpes = append(sharpes, s)
		}
	}

	result.Status = "completed"
	result.AggMetrics = map[string]float64{
		"windows":    float64(len(result.Windows)),
		"avg_return": mean(allReturns),
		"avg_sharpe": mean(sharpes),
	}
	return result, nil
}

type backtestWindow struct{ trainStart, trainEnd, testStart, testEnd string }

func buildWindows(cfg BacktestConfig) ([]backtestWindow, error) {
	start, _ := time.Parse("2006-01-02", cfg.StartDate)
	end, _ := time.Parse("2006-01-02", cfg.EndDate)
	if start.IsZero() || end.IsZero() {
		return nil, fmt.Errorf("backtest: invalid date range")
	}

	trainD := time.Duration(cfg.TrainWindow) * 24 * time.Hour
	testD := time.Duration(cfg.TestWindow) * 24 * time.Hour
	stepD := time.Duration(cfg.StepSize) * 24 * time.Hour
	if stepD <= 0 {
		stepD = testD
	}

	var windows []backtestWindow
	for cursor := start; cursor.Add(trainD + testD).Before(end.Add(24 * time.Hour)); cursor = cursor.Add(stepD) {
		ts := cursor
		te := cursor.Add(trainD)
		tts := te
		tte := tts.Add(testD)
		if tte.After(end.Add(24 * time.Hour)) {
			break
		}
		windows = append(windows, backtestWindow{
			trainStart: ts.Format("2006-01-02"), trainEnd: te.Format("2006-01-02"),
			testStart: tts.Format("2006-01-02"), testEnd: tte.Format("2006-01-02"),
		})
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("backtest: zero windows for %s→%s", cfg.StartDate, cfg.EndDate)
	}
	return windows, nil
}

func extractMetrics(r *ExecutionResult) map[string]float64 {
	m := map[string]float64{}
	for _, nr := range r.NodeResults {
		if nr.Status != "completed" || nr.Outputs == nil {
			continue
		}
		for _, key := range []string{"total_return", "sharpe", "max_drawdown", "win_rate"} {
			if v, ok := floatVal(nr.Outputs[key]); ok {
				m[key] = v
			}
		}
	}
	return m
}

func floatVal(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	}
	return 0, false
}

// ── Parameter Optimization ────────────────────────────────────────────

// OptimizeConfig defines a parameter sweep.
type OptimizeConfig struct {
	ParamSpace map[string][]any `json:"param_space"`
	Objective  string           `json:"objective"` // "sharpe" | "total_return" | "calmar"
	MaxEvals   int              `json:"max_evals"`
}

type OptimizeTrialResult struct {
	Params   map[string]any    `json:"params"`
	Metrics  map[string]float64 `json:"metrics"`
	Duration time.Duration     `json:"duration"`
	Error    string            `json:"error,omitempty"`
}

type OptimizeResult struct {
	Status     string               `json:"status"`
	Config     OptimizeConfig       `json:"config"`
	Trials     []OptimizeTrialResult `json:"trials"`
	TopParams  map[string]any       `json:"top_params"`
	TopMetrics map[string]float64   `json:"top_metrics"`
	StartedAt  time.Time            `json:"started_at"`
	FinishedAt time.Time            `json:"finished_at"`
}

// OptimizeParams runs grid search over the parameter space.
func (e *Engine) OptimizeParams(ctx context.Context, wf *Workflow, cfg OptimizeConfig) (*OptimizeResult, error) {
	result := &OptimizeResult{Status: "running", Config: cfg, StartedAt: time.Now()}
	defer func() { result.FinishedAt = time.Now() }()

	combos := generateCombos(cfg.ParamSpace)
	if cfg.MaxEvals > 0 && len(combos) > cfg.MaxEvals {
		combos = combos[:cfg.MaxEvals]
	}

	bestScore := math.Inf(-1)
	for i, combo := range combos {
		slog.Info("optimize: trial", "idx", i, "params", fmt.Sprint(combo))
		start := time.Now()

		trialWf := wf.Clone()
		for j := range trialWf.Nodes {
			if trialWf.Nodes[j].Params == nil {
				trialWf.Nodes[j].Params = make(map[string]any)
			}
			for k, v := range combo {
				trialWf.Nodes[j].Params[k] = v
			}
		}

		execResult, execErr := e.Execute(ctx, trialWf)
		duration := time.Since(start)
		trial := OptimizeTrialResult{Params: combo, Duration: duration}
		if execErr != nil {
			trial.Error = execErr.Error()
		} else {
			trial.Metrics = extractMetrics(execResult)
		}
		result.Trials = append(result.Trials, trial)

		score := getScore(trial.Metrics, cfg.Objective)
		if score > bestScore {
			bestScore = score
			result.TopParams = combo
			result.TopMetrics = trial.Metrics
		}
	}

	sort.Slice(result.Trials, func(i, j int) bool {
		return getScore(result.Trials[i].Metrics, cfg.Objective) > getScore(result.Trials[j].Metrics, cfg.Objective)
	})

	result.Status = "completed"
	return result, nil
}

func generateCombos(space map[string][]any) []map[string]any {
	if len(space) == 0 {
		return nil
	}
	var keys []string
	for k := range space {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var result []map[string]any
	var rec func(int, map[string]any)
	rec = func(idx int, cur map[string]any) {
		if idx == len(keys) {
			cp := make(map[string]any, len(cur))
			for k, v := range cur {
				cp[k] = v
			}
			result = append(result, cp)
			return
		}
		for _, val := range space[keys[idx]] {
			cur[keys[idx]] = val
			rec(idx+1, cur)
		}
	}
	rec(0, make(map[string]any))
	return result
}

func getScore(m map[string]float64, obj string) float64 {
	switch obj {
	case "sharpe":
		return m["sharpe"]
	case "total_return":
		return m["total_return"]
	case "calmar":
		if dd, ok := m["max_drawdown"]; ok && dd != 0 {
			return m["total_return"] / math.Abs(dd)
		}
		return 0
	default:
		return m["sharpe"]
	}
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
