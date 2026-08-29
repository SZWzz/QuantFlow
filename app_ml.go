package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"quantflow/internal/market"
	"quantflow/internal/python"
	"strings"
	"time"

	pb "quantflow/internal/python/proto"
)

// ── ML Model Registry ──────────────────────────────────────────────────

// MLModelInfo mirrors the frontend MLModel TypeScript interface.
type MLModelInfo struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	ModelType   string             `json:"model_type"`
	Category    string             `json:"category"`
	Hyperparams map[string]string  `json:"hyperparams"`
	Metrics     map[string]float64 `json:"metrics"`
	FilePath    string             `json:"file_path"`
	Status      string             `json:"status"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

// PredictionResult mirrors frontend Prediction.
type PredictionResult struct {
	ID         string   `json:"id"`
	ModelID    string   `json:"model_id"`
	Symbol     string   `json:"symbol"`
	Date       string   `json:"date"`
	Prediction float64  `json:"prediction"`
	Actual     *float64 `json:"actual"`
}

// DiscoveredFactorInfo mirrors frontend DiscoveredFactor.
type DiscoveredFactorInfo struct {
	Formula string  `json:"formula"`
	IC      float64 `json:"ic"`
	IR      float64 `json:"ir"`
	Sharpe  float64 `json:"sharpe"`
}

// ListMLModels scans the Python sidecar models/ directory for saved model
// artifacts. Returns empty slice when the directory does not exist.
func (a *App) ListMLModels() ([]MLModelInfo, error) {
	modelsDir := a.modelsDir()
	if modelsDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list_models: %w", err)
	}

	models := make([]MLModelInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".pkl") && !strings.HasSuffix(name, ".pt") &&
			!strings.HasSuffix(name, ".joblib") && !strings.HasSuffix(name, ".json") {
			continue
		}
		info, _ := entry.Info()
		ts := time.Now().Format(time.RFC3339)
		if info != nil {
			ts = info.ModTime().Format(time.RFC3339)
		}
		mt := detectModelType(name)
		models = append(models, MLModelInfo{
			ID:        strings.TrimSuffix(name, filepath.Ext(name)),
			Name:      strings.TrimSuffix(name, filepath.Ext(name)),
			ModelType: mt,
			Category:  detectCategory(mt),
			FilePath:  filepath.Join(modelsDir, name),
			Status:    "ready",
			CreatedAt: ts,
			UpdatedAt: ts,
		})
	}

	slog.Debug("list_models: scanned", "dir", modelsDir, "count", len(models))
	return models, nil
}

func (a *App) modelsDir() string {
	if a.bridge == nil {
		return ""
	}
	pyDir := a.bridge.PythonDir()
	if pyDir == "" {
		return ""
	}
	return filepath.Join(pyDir, "models")
}

func detectModelType(name string) string {
	l := strings.ToLower(name)
	switch {
	case strings.Contains(l, "xgboost") || strings.Contains(l, "xgb"):
		return "xgboost"
	case strings.Contains(l, "lightgbm") || strings.Contains(l, "lgb"):
		return "lightgbm"
	case strings.Contains(l, "catboost"):
		return "catboost"
	case strings.Contains(l, "randomforest") || strings.Contains(l, "rf"):
		return "random_forest"
	case strings.Contains(l, "lstm"):
		return "lstm"
	case strings.Contains(l, "transformer"):
		return "transformer"
	case strings.Contains(l, "ppo") || strings.Contains(l, "dqn") || strings.Contains(l, "rl"):
		return "rl"
	default:
		return "unknown"
	}
}

func detectCategory(mt string) string {
	switch mt {
	case "xgboost", "lightgbm", "catboost", "random_forest", "lstm", "transformer":
		return "prediction"
	case "rl":
		return "reinforcement_learning"
	default:
		return "other"
	}
}

// ── Predictions ────────────────────────────────────────────────────────

// GetPredictions returns predictions for a model+symbol pair via Python gRPC.
func (a *App) GetPredictions(modelID, symbol string) ([]PredictionResult, error) {
	if a.bridge == nil {
		return nil, nil
	}
	client := python.NewMLClient(a.bridge)
	req := &pb.PredictRequest{ModelId: modelID}
	resp, err := client.Predict(context.Background(), req)
	if err != nil {
		slog.Warn("get_predictions: failed", "model", modelID, "error", err)
		return nil, nil
	}

	today := time.Now().Format("2006-01-02")
	results := make([]PredictionResult, 0, len(resp.Predictions))
	for i, p := range resp.Predictions {
		results = append(results, PredictionResult{
			ID:         fmt.Sprintf("%s-%s-%d", modelID, today, i),
			ModelID:    modelID,
			Symbol:     symbol,
			Date:       today,
			Prediction: p,
		})
	}
	return results, nil
}

// ── Alpha Mining ───────────────────────────────────────────────────────

// AlphaMiningParams carries genetic-programming factor discovery config.
type AlphaMiningParams struct {
	FactorNames    []string `json:"factor_names"`
	PopulationSize int32    `json:"population_size"`
	Generations    int32    `json:"generations"`
	CrossoverRate  float64  `json:"crossover_rate"`
	MutationRate   float64  `json:"mutation_rate"`
	TopK           int32    `json:"top_k"`
	FitnessMetric  string   `json:"fitness_metric"`
}

// RunAlphaMining triggers factor discovery via Python gRPC AlphaMining.
func (a *App) RunAlphaMining(params AlphaMiningParams) ([]DiscoveredFactorInfo, error) {
	if a.bridge == nil {
		return nil, fmt.Errorf("Python sidecar not available")
	}
	client := python.NewMLClient(a.bridge)
	req := &pb.AlphaMiningRequest{
		BaseFactorNames: params.FactorNames,
		PopulationSize:  params.PopulationSize,
		Generations:     params.Generations,
		CrossoverRate:   params.CrossoverRate,
		MutationRate:    params.MutationRate,
		TopK:            params.TopK,
		FitnessMetric:   params.FitnessMetric,
	}
	resp, err := client.AlphaMining(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("alpha_mining: %w", err)
	}
	factors := make([]DiscoveredFactorInfo, 0, len(resp.Factors))
	for _, f := range resp.Factors {
		factors = append(factors, DiscoveredFactorInfo{
			Formula: f.Formula, IC: f.Ic, IR: f.Ir, Sharpe: f.Sharpe,
		})
	}
	slog.Info("alpha_mining: done", "found", len(factors), "ms", resp.MiningTimeMs)
	return factors, nil
}

// ── Risk Modeling ──────────────────────────────────────────────────────

// AssessRisk runs a risk model via Python gRPC RiskModel.
// It fetches 252 days of OHLCV data, computes daily log returns per symbol,
// and sends them to the Python sidecar for GARCH/covariance estimation.
func (a *App) AssessRisk(symbols []string, modelType string) (map[string]interface{}, error) {
	if a.bridge == nil {
		return nil, nil
	}

	// Compute daily returns for each symbol from OHLCV data.
	returnsMatrix := a.computeReturnsForRiskModel(symbols)
	returnsJSON, err := json.Marshal(returnsMatrix)
	if err != nil {
		slog.Warn("assess_risk: marshal returns", "error", err)
		returnsJSON = []byte("[]")
	}

	client := python.NewMLClient(a.bridge)
	req := &pb.RiskModelRequest{
		ModelType:   modelType,
		ReturnsData: returnsJSON,
		Params:      map[string]string{"symbols": strings.Join(symbols, ",")},
	}
	resp, err := client.RiskModel(context.Background(), req)
	if err != nil {
		slog.Warn("assess_risk: failed", "model", modelType, "error", err)
		return nil, nil
	}
	result := map[string]interface{}{
		"model_type":      modelType,
		"compute_time_ms": resp.ComputeTimeMs,
	}
	for k, v := range resp.Metrics {
		result[k] = v
	}
	return result, nil
}

// computeReturnsForRiskModel fetches OHLCV data for the given symbols and
// computes daily log returns. Returns a [][]float64 matrix (symbols × days).
func (a *App) computeReturnsForRiskModel(symbols []string) [][]float64 {
	end := time.Now()
	start := end.AddDate(0, 0, -365) // ~252 trading days

	matrix := make([][]float64, 0, len(symbols))
	for _, sym := range symbols {
		bars, err := a.fetchOHLCVForSymbol(sym, start, end)
		if err != nil || len(bars) < 2 {
			slog.Warn("assess_risk: skip symbol, insufficient OHLCV data", "symbol", sym)
			continue
		}
		returns := make([]float64, 0, len(bars)-1)
		for i := 1; i < len(bars); i++ {
			if bars[i-1].Close > 0 {
				r := (bars[i].Close - bars[i-1].Close) / bars[i-1].Close
				returns = append(returns, r)
			}
		}
		if len(returns) > 0 {
			matrix = append(matrix, returns)
		}
	}
	return matrix
}

// fetchOHLCVForSymbol resolves a symbol's market and fetches daily OHLCV bars.
func (a *App) fetchOHLCVForSymbol(symbol string, start, end time.Time) ([]market.OHLCVBar, error) {
	if a.marketReg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	marketName := market.MarketForSymbol(symbol)
	bars, _, err := a.marketReg.FetchOHLCVWithFallback(context.Background(),
		marketName, symbol, "1D", "", start.Unix(), end.Unix())
	return bars, err
}
