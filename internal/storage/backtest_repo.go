package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// StoredBacktest represents a persisted backtest run with full data including
// JSON blobs for config, equity curve, trades, and OHLCV data.
type StoredBacktest struct {
	ID           int     `json:"id"`
	RunID        string  `json:"run_id"`
	WorkflowName string  `json:"workflow_name"`
	StrategyName string  `json:"strategy_name"`
	Symbol       string  `json:"symbol"`
	EngineType   string  `json:"engine_type"`
	TotalReturn  float64 `json:"total_return"`
	CAGR         float64 `json:"cagr"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	SortinoRatio float64 `json:"sortino_ratio"`
	CalmarRatio  float64 `json:"calmar_ratio"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
	TotalTrades  int     `json:"total_trades"`
	ConfigJSON   string  `json:"config_json"`
	EquityCurve  string  `json:"equity_curve"`
	TradesJSON   string  `json:"trades_json"`
	OHLCVData    string  `json:"ohlcv_data"`
	StartedAt    string  `json:"started_at"`
	FinishedAt   string  `json:"finished_at"`
	CreatedAt    string  `json:"created_at"`
}

// StoredBacktestSummary contains metadata and performance metrics for a
// backtest run, without the large JSON blobs.
type StoredBacktestSummary struct {
	ID           int     `json:"id"`
	RunID        string  `json:"run_id"`
	WorkflowName string  `json:"workflow_name"`
	StrategyName string  `json:"strategy_name"`
	Symbol       string  `json:"symbol"`
	EngineType   string  `json:"engine_type"`
	TotalReturn  float64 `json:"total_return"`
	CAGR         float64 `json:"cagr"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	SortinoRatio float64 `json:"sortino_ratio"`
	CalmarRatio  float64 `json:"calmar_ratio"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
	TotalTrades  int     `json:"total_trades"`
	StartedAt    string  `json:"started_at"`
	FinishedAt   string  `json:"finished_at"`
	CreatedAt    string  `json:"created_at"`
}

// BacktestRepo provides CRUD operations for backtest result persistence.
type BacktestRepo struct {
	db *sql.DB
}

// NewBacktestRepo creates a new backtest result repository.
func NewBacktestRepo(db *sql.DB) *BacktestRepo {
	return &BacktestRepo{db: db}
}

// Save persists a new backtest result. Returns the auto-generated ID.
func (r *BacktestRepo) Save(ctx context.Context, bt StoredBacktest) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO backtest_results (
			run_id, workflow_name, strategy_name, symbol, engine_type,
			total_return, cagr, max_drawdown, sharpe_ratio, sortino_ratio,
			calmar_ratio, win_rate, profit_factor, total_trades,
			config_json, equity_curve, trades_json, ohlcv_data,
			started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bt.RunID, bt.WorkflowName, bt.StrategyName, bt.Symbol, bt.EngineType,
		bt.TotalReturn, bt.CAGR, bt.MaxDrawdown, bt.SharpeRatio, bt.SortinoRatio,
		bt.CalmarRatio, bt.WinRate, bt.ProfitFactor, bt.TotalTrades,
		bt.ConfigJSON, bt.EquityCurve, bt.TradesJSON, bt.OHLCVData,
		bt.StartedAt, bt.FinishedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("save backtest result: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	return int(id), nil
}

// List returns recent backtest results ordered by finished_at descending,
// with pagination via limit and offset.
func (r *BacktestRepo) List(ctx context.Context, limit, offset int) ([]StoredBacktestSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, run_id, workflow_name, strategy_name, symbol, engine_type,
			total_return, cagr, max_drawdown, sharpe_ratio, sortino_ratio,
			calmar_ratio, win_rate, profit_factor, total_trades,
			started_at, finished_at, created_at
		 FROM backtest_results ORDER BY finished_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list backtest results: %w", err)
	}
	defer rows.Close()
	return scanBacktestSummaries(rows)
}

// GetByID returns a single backtest result by its primary key.
func (r *BacktestRepo) GetByID(ctx context.Context, id int) (*StoredBacktest, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, run_id, workflow_name, strategy_name, symbol, engine_type,
			total_return, cagr, max_drawdown, sharpe_ratio, sortino_ratio,
			calmar_ratio, win_rate, profit_factor, total_trades,
			config_json, equity_curve, trades_json, ohlcv_data,
			started_at, finished_at, created_at
		 FROM backtest_results WHERE id=?`, id,
	)
	bt := &StoredBacktest{}
	err := row.Scan(
		&bt.ID, &bt.RunID, &bt.WorkflowName, &bt.StrategyName, &bt.Symbol, &bt.EngineType,
		&bt.TotalReturn, &bt.CAGR, &bt.MaxDrawdown, &bt.SharpeRatio, &bt.SortinoRatio,
		&bt.CalmarRatio, &bt.WinRate, &bt.ProfitFactor, &bt.TotalTrades,
		&bt.ConfigJSON, &bt.EquityCurve, &bt.TradesJSON, &bt.OHLCVData,
		&bt.StartedAt, &bt.FinishedAt, &bt.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backtest result %d: %w", id, err)
	}
	return bt, nil
}

// Delete removes a backtest result by its primary key.
func (r *BacktestRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM backtest_results WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete backtest result %d: %w", id, err)
	}
	return nil
}

// scanBacktestSummaries scans all rows from a summary query.
func scanBacktestSummaries(rows *sql.Rows) ([]StoredBacktestSummary, error) {
	var summaries []StoredBacktestSummary
	for rows.Next() {
		var s StoredBacktestSummary
		if err := rows.Scan(
			&s.ID, &s.RunID, &s.WorkflowName, &s.StrategyName, &s.Symbol, &s.EngineType,
			&s.TotalReturn, &s.CAGR, &s.MaxDrawdown, &s.SharpeRatio, &s.SortinoRatio,
			&s.CalmarRatio, &s.WinRate, &s.ProfitFactor, &s.TotalTrades,
			&s.StartedAt, &s.FinishedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan backtest summary row: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}
