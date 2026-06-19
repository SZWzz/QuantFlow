package trading

import (
	"context"
	"log/slog"
)

// Engine is the main trading engine that coordinates the bar-by-bar pipeline.
type Engine struct {
	paperEngine *PaperEngine
	signalCh    chan Signal
	barCh       chan OHLCVBar
	done        chan struct{}
}

// NewEngine creates a new trading engine with paper trading and risk management.
func NewEngine(initialCapital float64) *Engine {
	oms := NewOMS()
	riskConfig := DefaultRiskConfig()
	paperEngine := NewPaperEngine(oms, riskConfig, initialCapital)

	return &Engine{
		paperEngine: paperEngine,
		signalCh:    make(chan Signal, 256),
		barCh:       make(chan OHLCVBar, 1024),
		done:        make(chan struct{}),
	}
}

// Start begins the engine's main loop.
func (e *Engine) Start(ctx context.Context) {
	slog.Info("trading engine started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("trading engine shutting down")
			close(e.done)
			return

		case signal := <-e.signalCh:
			slog.Debug("processing signal", "symbol", signal.Symbol, "direction", signal.Direction)
			var side OrderSide
			if signal.Direction == "sell" {
				side = SideSell
			} else {
				side = SideBuy
			}

			orderType := TypeMarket
			price := signal.Price
			if price > 0 {
				orderType = TypeLimit
			}

			_, err := e.paperEngine.PlaceOrder(signal.Symbol, side, orderType, signal.Quantity, price)
			if err != nil {
				slog.Error("failed to place order from signal", "error", err)
			}

		case bar := <-e.barCh:
			trades := e.paperEngine.OnBar(bar)
			if len(trades) > 0 {
				slog.Info("trades executed", "count", len(trades), "symbol", bar.Symbol)
			}
		}
	}
}

// SubmitSignal sends a trading signal to the engine.
func (e *Engine) SubmitSignal(signal Signal) {
	select {
	case e.signalCh <- signal:
	default:
		slog.Warn("signal channel full, dropping signal", "symbol", signal.Symbol)
	}
}

// SubmitBar sends a price bar to the engine for order matching.
func (e *Engine) SubmitBar(bar OHLCVBar) {
	select {
	case e.barCh <- bar:
	default:
		slog.Warn("bar channel full, dropping bar", "symbol", bar.Symbol)
	}
}

// GetPaperEngine returns the underlying paper trading engine.
func (e *Engine) GetPaperEngine() *PaperEngine {
	return e.paperEngine
}

// Done returns a channel that is closed when the engine shuts down.
func (e *Engine) Done() <-chan struct{} {
	return e.done
}
