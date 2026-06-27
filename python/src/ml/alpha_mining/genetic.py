"""Genetic programming engine for alpha factor discovery using gplearn."""
import logging
import numpy as np
import pandas as pd
import pyarrow as pa

logger = logging.getLogger(__name__)

_HAS_GPLEARN = False
try:
    from gplearn.genetic import SymbolicRegressor
    from gplearn.functions import make_function
    _HAS_GPLEARN = True
except ImportError:
    pass


class AlphaMiningEngine:
    """Evolves alpha factor formulas from a pool of base factors using genetic programming."""

    def __init__(self):
        self._check_gplearn()

    def _check_gplearn(self):
        if not _HAS_GPLEARN:
            raise ImportError(
                "gplearn is required for AlphaMining. Install with: pip install gplearn"
            )

    def evolve(self, factor_data: pa.Table, returns_data: pa.Table, params: dict) -> list[dict]:
        """Run genetic programming to discover predictive factor formulas.

        Args:
            factor_data: Arrow Table where each column is a base factor series.
            returns_data: Arrow Table with 'return' column (forward returns for IC computation).
            params: Configuration dict with keys: population_size, generations, top_k,
                    crossover_rate, mutation_rate, fitness_metric.

        Returns:
            List of dicts with keys: formula, ic, ir, sharpe (sorted by |IC| descending).
        """
        self._check_gplearn()

        pop_size = int(params.get("population_size", 200))
        generations = int(params.get("generations", 50))
        top_k = int(params.get("top_k", 20))
        crossover_rate = float(params.get("crossover_rate", 0.7))
        mutation_rate = float(params.get("mutation_rate", 0.1))
        fitness_metric = params.get("fitness_metric", "ic")

        # Convert Arrow to numpy
        df_factors = factor_data.to_pandas()
        X = df_factors.values.astype(np.float64)
        y = returns_data.column("return").to_numpy().astype(np.float64)
        col_names = df_factors.columns.tolist()

        # Clean data
        mask = ~np.isnan(y) & ~np.any(np.isnan(X), axis=1)
        X, y = X[mask], y[mask]

        if len(y) < 100:
            raise ValueError(f"not enough valid samples: {len(y)}")

        # Time-series train/test split (no shuffle)
        split_idx = int(len(X) * 0.7)
        X_train, X_test = X[:split_idx], X[split_idx:]
        y_train, y_test = y[:split_idx], y[split_idx:]

        # Build custom function set from factor names
        function_set = ['add', 'sub', 'mul', 'div', 'sqrt', 'log', 'abs', 'neg', 'inv',
                        'sin', 'cos', 'tan']

        # Train SymbolicRegressor
        gp = SymbolicRegressor(
            population_size=pop_size,
            generations=generations,
            tournament_size=20,
            stopping_criteria=-1.0,
            const_range=(-1.0, 1.0),
            init_depth=(2, 6),
            function_set=function_set,
            metric='pearson' if fitness_metric == 'ic' else 'mse',
            parsimony_coefficient=0.001,
            p_crossover=crossover_rate,
            p_subtree_mutation=0.1,
            p_hoist_mutation=0.05,
            p_point_mutation=0.1,
            p_point_replace=mutation_rate,
            max_samples=0.9,
            verbose=0,
            random_state=42,
            n_jobs=1,
        )
        gp.fit(X_train, y_train)

        # Extract discovered formulas (from the Pareto front / best programs)
        results = []
        seen = set()

        # Evaluate each program in the final population
        for i, program in enumerate(gp._programs[-1]):
            if len(results) >= top_k:
                break

            formula_str = str(program)
            # Map variable names (X0, X1, ...) back to factor column names
            for j, name in enumerate(col_names):
                formula_str = formula_str.replace(f"X{j}", name)

            if formula_str in seen:
                continue
            seen.add(formula_str)

            y_pred = program.execute(X_test)
            if np.any(np.isnan(y_pred)):
                continue

            # Compute IC (Pearson correlation) on test set
            ic = float(np.corrcoef(y_pred, y_test)[0, 1]) if np.std(y_pred) > 0 else 0.0
            if np.isnan(ic):
                ic = 0.0

            # Compute IR (IC / std of rolling IC) on test set
            n_rolling = min(20, len(y_test) // 5)
            rolling_ics = []
            for start in range(0, len(y_test) - n_rolling, n_rolling):
                end = start + n_rolling
                if np.std(y_pred[start:end]) > 0:
                    ric = np.corrcoef(y_pred[start:end], y_test[start:end])[0, 1]
                    if not np.isnan(ric):
                        rolling_ics.append(ric)
            ic_std = np.std(rolling_ics) if len(rolling_ics) > 1 else 1.0
            ir = float(abs(ic) / ic_std) if ic_std > 0 else 0.0

            # Compute Sharpe via quantile backtest on test set
            sharpe = self._quantile_sharpe(y_pred, y_test)

            results.append({
                "formula": formula_str,
                "ic": round(ic, 6),
                "ir": round(ir, 4),
                "sharpe": round(sharpe, 4),
            })

        # Sort by |IC| descending
        results.sort(key=lambda r: abs(r["ic"]), reverse=True)
        return results[:top_k]

    def _quantile_sharpe(self, predictions: np.ndarray, returns: np.ndarray, n_quantiles: int = 5) -> float:
        """Estimate Sharpe via long-short quantile strategy."""
        try:
            quantiles = np.percentile(predictions, np.linspace(0, 100, n_quantiles + 1))
            top_mask = predictions >= quantiles[-2]
            bot_mask = predictions <= quantiles[1]
            long_ret = returns[top_mask].mean() if top_mask.sum() > 0 else 0
            short_ret = -returns[bot_mask].mean() if bot_mask.sum() > 0 else 0
            spread = long_ret + short_ret
            spread_std = np.std(returns[top_mask | bot_mask]) if (top_mask | bot_mask).sum() > 1 else 0.01
            return float(spread / spread_std * np.sqrt(252)) if spread_std > 0 else 0.0
        except Exception:
            return 0.0
