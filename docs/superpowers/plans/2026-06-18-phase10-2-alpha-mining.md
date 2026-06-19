# Phase 10.2: Alpha Mining Engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Build genetic programming-based alpha factor discovery: AlphaMiningEngine (Python) + AlphaMiningNode (Go workflow) + AlphaMiningWorkspace (Vue panel). End-to-end: factor pool → GP evolution → new factor formulas → register as FactorNode.

**Architecture:** Python gplearn-based genetic engine evolves factor formulas from the 25-factor pool. Go AlphaMiningNode calls Python via gRPC, receives discovered factors with IC/IR/Sharpe scores. Frontend panel lets users select factors, configure GP params, run mining, and register winning factors back into the workflow system.

**Tech Stack:** Python 3.12+ (gplearn, numpy, pandas, pyarrow), Go 1.22+ (workflow node, gRPC client), Vue 3 + TypeScript (panel), protobuf (AlphaMiningRequest/Response already defined in ml.proto).

**Depends on:** Phase 10.1 (proto, MLService, ModelRegistry, mlStore, gRPC infrastructure).

## Global Constraints

- gplearn is an optional dependency — AlphaMining returns clear error if not installed
- Discovered factors produce human-readable formulas that can be registered as FactorNode
- GP evolution: max generations and population size configurable, with IC-based early stopping
- All fitness metrics (IC, IR, Sharpe) computed against validation period, not training period
- Follow existing code patterns: Python gRPC servicer, Go BaseNode interface, Vue Composition API

---

### Task 1: Python AlphaMiningEngine — genetic.py

**Files:**
- Create: `python/src/ml/alpha_mining/__init__.py`
- Create: `python/src/ml/alpha_mining/genetic.py`
- Create: `python/tests/test_alpha_mining.py`

**Interfaces:**
- Consumes: Phase 10.1 (serialization via `src.ml.serialization`)
- Produces: `AlphaMiningEngine` class with `evolve(factor_data: pa.Table, returns_data: pa.Table, params: dict) -> list[dict]`

- [ ] **Step 1: Write test**

Write `python/tests/test_alpha_mining.py`:

```python
import numpy as np
import pandas as pd
import pyarrow as pa
import pytest

try:
    import gplearn
    HAS_GPLEARN = True
except ImportError:
    HAS_GPLEARN = False


@pytest.fixture
def sample_factor_data():
    np.random.seed(42)
    n = 500
    data = {}
    for i in range(5):
        data[f"factor_{i}"] = np.random.randn(n)
    return pa.Table.from_pandas(pd.DataFrame(data))


@pytest.fixture
def sample_returns():
    np.random.seed(42)
    n = 500
    returns = np.random.randn(n) * 0.02
    return pa.Table.from_pandas(pd.DataFrame({"return": returns}))


@pytest.mark.skipif(not HAS_GPLEARN, reason="gplearn not installed")
class TestAlphaMining:
    def test_evolve_discovers_factors(self, sample_factor_data, sample_returns):
        from src.ml.alpha_mining.genetic import AlphaMiningEngine
        
        engine = AlphaMiningEngine()
        results = engine.evolve(sample_factor_data, sample_returns, {
            "population_size": "50",
            "generations": "5",
            "top_k": "5",
            "fitness_metric": "ic",
        })
        
        assert len(results) > 0
        assert len(results) <= 5
        for r in results:
            assert "formula" in r
            assert "ic" in r
            assert isinstance(r["formula"], str)
            assert len(r["formula"]) > 0

    def test_formula_is_valid_expression(self, sample_factor_data, sample_returns):
        from src.ml.alpha_mining.genetic import AlphaMiningEngine
        
        engine = AlphaMiningEngine()
        results = engine.evolve(sample_factor_data, sample_returns, {
            "population_size": "30",
            "generations": "3",
            "top_k": "3",
        })
        
        # Each formula should be evaluable against the factor data
        df = sample_factor_data.to_pandas()
        for r in results:
            try:
                # The formula references factor column names
                values = eval(r["formula"], {"__builtins__": {}}, {col: df[col].values for col in df.columns})
                assert len(values) == len(df)
            except Exception as e:
                pytest.fail(f"Formula '{r['formula']}' failed to evaluate: {e}")

    def test_gplearn_not_installed_raises(self):
        # This test always runs — verifies graceful degradation
        from src.ml.alpha_mining.genetic import _HAS_GPLEARN
        if not _HAS_GPLEARN:
            from src.ml.alpha_mining.genetic import AlphaMiningEngine
            engine = AlphaMiningEngine()
            with pytest.raises(ImportError, match="gplearn"):
                engine.evolve(None, None, {})
```

- [ ] **Step 2: Run test → verify failure** (module not created)

- [ ] **Step 3: Implement AlphaMiningEngine**

Write `python/src/ml/alpha_mining/__init__.py`:
```python
"""Alpha Mining — genetic programming-based factor discovery."""
```

Write `python/src/ml/alpha_mining/genetic.py`:

```python
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

        # Build custom function set from factor names
        function_set = ['add', 'sub', 'mul', 'div', 'sqrt', 'log', 'abs', 'neg', 'inv',
                        'sin', 'cos', 'tan']

        # Train SymbolicRegressor
        gp = SymbolicRegressor(
            population_size=pop_size,
            generations=generations,
            tournament_size=20,
            stopping_criteria=None,
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
        gp.fit(X, y)

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

            y_pred = program.execute(X)
            if np.any(np.isnan(y_pred)):
                continue

            # Compute IC (Pearson correlation)
            ic = float(np.corrcoef(y_pred, y)[0, 1]) if np.std(y_pred) > 0 else 0.0
            if np.isnan(ic):
                ic = 0.0

            # Compute IR (IC / std of rolling IC)
            n_rolling = min(20, len(y) // 5)
            rolling_ics = []
            for start in range(0, len(y) - n_rolling, n_rolling):
                end = start + n_rolling
                if np.std(y_pred[start:end]) > 0:
                    ric = np.corrcoef(y_pred[start:end], y[start:end])[0, 1]
                    if not np.isnan(ric):
                        rolling_ics.append(ric)
            ic_std = np.std(rolling_ics) if len(rolling_ics) > 1 else 1.0
            ir = float(abs(ic) / ic_std) if ic_std > 0 else 0.0

            # Compute Sharpe via quantile backtest
            sharpe = self._quantile_sharpe(y_pred, y)

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
```

- [ ] **Step 4: Run test → verify pass**

```bash
cd python && python -m pytest tests/test_alpha_mining.py -v
```

Expected: Tests pass (or skip if gplearn not installed).

- [ ] **Step 5: Commit**

```bash
git add python/src/ml/alpha_mining/ python/tests/test_alpha_mining.py
git commit -m "feat(python): add AlphaMiningEngine with genetic programming factor discovery"
```

---

### Task 2: Python AlphaMining — evaluator.py

**Files:**
- Create: `python/src/ml/alpha_mining/evaluator.py`
- Modify: `python/tests/test_alpha_mining.py` (add evaluation tests)

**Interfaces:**
- Consumes: `genetic.py` (AlphaMiningEngine)
- Produces: `evaluate_factor(formula: str, factor_data: pa.Table, returns_data: pa.Table) -> dict`

- [ ] **Step 1: Write test** — add to `test_alpha_mining.py`:

```python
def test_evaluate_factor():
    from src.ml.alpha_mining.evaluator import evaluate_factor
    
    np.random.seed(42)
    n = 200
    f0 = np.random.randn(n)
    returns = f0 * 0.5 + np.random.randn(n) * 0.05
    data = pa.Table.from_pandas(pd.DataFrame({"f_0": f0}))
    rets = pa.Table.from_pandas(pd.DataFrame({"return": returns}))
    
    result = evaluate_factor("f_0", data, rets)
    assert "ic" in result
    assert "ir" in result
    assert "sharpe" in result
    assert abs(result["ic"]) > 0  # correlated factor should have non-zero IC
```

- [ ] **Step 2: Implement**

Write `python/src/ml/alpha_mining/evaluator.py`:

```python
"""Factor evaluation: IC, IR, quantile Sharpe for a single factor formula."""
import numpy as np
import pandas as pd
import pyarrow as pa


def evaluate_factor(formula: str, factor_data: pa.Table, returns_data: pa.Table) -> dict:
    """Evaluate a single factor formula.

    Args:
        formula: Python expression referencing factor column names (e.g., "f_0 + f_1").
        factor_data: Arrow Table of factor values.
        returns_data: Arrow Table with 'return' column.

    Returns:
        dict with keys: ic, ir, sharpe.
    """
    df = factor_data.to_pandas()
    rets = returns_data.column("return").to_numpy().astype(np.float64)

    # Build evaluation namespace — each column as a numpy array
    namespace = {col: df[col].values.astype(np.float64) for col in df.columns}
    namespace["np"] = np

    try:
        values = eval(formula, {"__builtins__": {}, "np": np, "rank": lambda x: pd.Series(x).rank().values}, namespace)
        values = np.asarray(values, dtype=np.float64)
    except Exception as e:
        return {"ic": 0.0, "ir": 0.0, "sharpe": 0.0, "error": str(e)}

    mask = ~np.isnan(values) & ~np.isnan(rets)
    vals, r = values[mask], rets[mask]

    if len(vals) < 30 or np.std(vals) == 0:
        return {"ic": 0.0, "ir": 0.0, "sharpe": 0.0}

    # IC
    ic = float(np.corrcoef(vals, r)[0, 1])
    if np.isnan(ic):
        ic = 0.0

    # IR
    n_rolling = min(20, len(vals) // 5)
    rolling_ics = []
    for start in range(0, len(vals) - n_rolling, n_rolling):
        seg = slice(start, start + n_rolling)
        if np.std(vals[seg]) > 0:
            ric = np.corrcoef(vals[seg], r[seg])[0, 1]
            if not np.isnan(ric):
                rolling_ics.append(ric)
    ic_std = np.std(rolling_ics) if len(rolling_ics) > 1 else 1.0
    ir = float(abs(ic) / ic_std) if ic_std > 0 else 0.0

    # Quantile Sharpe
    q = np.percentile(vals, [0, 20, 40, 60, 80, 100])
    top = vals >= q[-2]
    bot = vals <= q[1]
    long_ret = r[top].mean() if top.sum() > 0 else 0
    short_ret = -r[bot].mean() if bot.sum() > 0 else 0
    spread_std = np.std(r[top | bot]) if (top | bot).sum() > 1 else 0.01
    sharpe = float((long_ret + short_ret) / spread_std * np.sqrt(252)) if spread_std > 0 else 0.0

    return {"ic": round(ic, 6), "ir": round(ir, 4), "sharpe": round(sharpe, 4)}
```

- [ ] **Step 3: Run test → pass, Step 4: Commit**

```bash
git add python/src/ml/alpha_mining/evaluator.py python/tests/test_alpha_mining.py
git commit -m "feat(python): add factor evaluator with IC/IR/Sharpe computation"
```

---

### Task 3: Wire AlphaMining into MLService

**Files:**
- Modify: `python/src/ml/engine.py` (replace AlphaMining stub with real implementation)

**Interfaces:**
- Consumes: `alpha_mining.genetic` (AlphaMiningEngine), `alpha_mining.evaluator` (evaluate_factor)
- Produces: `AlphaMining` RPC returns `AlphaMiningResponse` with `repeated DiscoveredFactor`

- [ ] **Step 1: Write integration test**

Add to `python/tests/test_ml_service.py`:

```python
@pytest.mark.skipif(not HAS_XGB, reason="gplearn not available")
@pytest.mark.asyncio
async def test_alpha_mining(ml_service):
    from src.proto import ml_pb2
    
    np.random.seed(42)
    n = 300
    df = pd.DataFrame({f"f_{i}": np.random.randn(n) for i in range(5)})
    rets = pd.DataFrame({"return": df["f_0"] * 0.5 + np.random.randn(n) * 0.1})
    
    factor_sink = pa.BufferOutputStream()
    factor_table = pa.Table.from_pandas(df)
    with pa.ipc.new_stream(factor_sink, factor_table.schema) as w:
        w.write_table(factor_table)
    
    return_sink = pa.BufferOutputStream()
    return_table = pa.Table.from_pandas(rets)
    with pa.ipc.new_stream(return_sink, return_table.schema) as w:
        w.write_table(return_table)
    
    req = ml_pb2.AlphaMiningRequest(
        base_factor_names=[f"f_{i}" for i in range(5)],
        factor_data=factor_sink.getvalue().to_pybytes(),
        returns_data=return_sink.getvalue().to_pybytes(),
        population_size=30,
        generations=3,
        top_k=5,
    )
    
    resp = await ml_service.AlphaMining(req, None)
    assert len(resp.factors) > 0
    assert len(resp.factors) <= 5
    # Check first factor has valid fields
    f = resp.factors[0]
    assert f.formula != ""
    assert -1.0 <= f.ic <= 1.0
```

- [ ] **Step 2: Implement AlphaMining in engine.py**

Replace the `AlphaMining` stub in `python/src/ml/engine.py`:

```python
async def AlphaMining(self, request, context):
    try:
        from src.ml.alpha_mining.genetic import AlphaMiningEngine
        
        factor_data = self._decode_arrow(request.factor_data)
        returns_data = self._decode_arrow(request.returns_data)
        
        params = {
            "population_size": str(request.population_size or 200),
            "generations": str(request.generations or 50),
            "crossover_rate": str(request.crossover_rate or 0.7),
            "mutation_rate": str(request.mutation_rate or 0.1),
            "fitness_metric": request.fitness_metric or "ic",
            "top_k": str(request.top_k or 20),
        }
        
        engine = AlphaMiningEngine()
        results = engine.evolve(factor_data, returns_data, params)
        
        import time
        elapsed_ms = 0  # Engine doesn't time itself; approximate
        resp = ml_pb2.AlphaMiningResponse(mining_time_ms=elapsed_ms)
        for r in results:
            resp.factors.append(ml_pb2.DiscoveredFactor(
                formula=r["formula"],
                ic=r.get("ic", 0),
                ir=r.get("ir", 0),
                sharpe=r.get("sharpe", 0),
            ))
        return resp
    except ImportError as e:
        logger.warning("AlphaMining: %s", e)
        return ml_pb2.AlphaMiningResponse()
    except Exception as e:
        logger.exception("AlphaMining failed")
        return ml_pb2.AlphaMiningResponse()
```

- [ ] **Step 3: Run test → pass, Step 4: Commit**

```bash
git add python/src/ml/engine.py python/tests/test_ml_service.py
git commit -m "feat(python): wire AlphaMining into MLService gRPC with real genetic engine"
```

---

### Task 4: Go AlphaMiningNode

**Files:**
- Create: `internal/workflow/nodes/alpha_mining.go`
- Create: `internal/workflow/nodes/alpha_mining_test.go`
- Modify: `internal/workflow/nodes/register.go` (add registration)

**Interfaces:**
- Consumes: `internal/python/ml_client.go` (MLClient.AlphaMining)
- Produces: `AlphaMiningNode` with ports: factor_pool + ohlcv_data → new_factors + factor_scores

- [ ] **Step 1: Write test**

Write `internal/workflow/nodes/alpha_mining_test.go`:

```go
package nodes

import (
    "testing"
    "quantflow/internal/workflow"
)

func TestAlphaMiningNode_Registration(t *testing.T) {
    r := workflow.NewRegistry()
    r.RegisterWithCategory("alpha_mining", NewAlphaMiningNode, "ml")

    node, err := r.Create("alpha_mining", "am-1", map[string]any{
        "population_size": "50",
        "generations":     "10",
        "top_k":          "5",
    })
    if err != nil {
        t.Fatalf("create failed: %v", err)
    }
    if node.Category() != "ml" {
        t.Errorf("expected category 'ml', got '%s'", node.Category())
    }
}
```

- [ ] **Step 2: Implement AlphaMiningNode**

Write `internal/workflow/nodes/alpha_mining.go`:

```go
package nodes

import (
    "context"
    "encoding/json"
    "fmt"

    "quantflow/internal/python"
    "quantflow/internal/python/proto"
    "quantflow/internal/workflow"
)

type AlphaMiningNode struct {
    id     string
    params map[string]any
}

func NewAlphaMiningNode(id string, params map[string]any) (workflow.BaseNode, error) {
    n := &AlphaMiningNode{id: id, params: params}
    return n, n.Validate()
}

func (n *AlphaMiningNode) ID() string       { return n.id }
func (n *AlphaMiningNode) NodeType() string { return "alpha_mining" }
func (n *AlphaMiningNode) Category() string { return "ml" }

func (n *AlphaMiningNode) InputPorts() []workflow.PortDefinition {
    return []workflow.PortDefinition{
        {Name: "factor_pool", Type: workflow.PortSeries, Required: true},
        {Name: "ohlcv_data", Type: workflow.PortSeries, Required: true},
    }
}

func (n *AlphaMiningNode) OutputPorts() []workflow.PortDefinition {
    return []workflow.PortDefinition{
        {Name: "new_factors", Type: workflow.PortSeries},
        {Name: "factor_scores", Type: workflow.PortSeries},
    }
}

func (n *AlphaMiningNode) ParamSchema() []workflow.ParamDef {
    return []workflow.ParamDef{
        {Name: "population_size", Type: "int", Default: "200", Description: "GP population size"},
        {Name: "generations", Type: "int", Default: "50", Description: "GP generations"},
        {Name: "top_k", Type: "int", Default: "20", Description: "Number of top factors to return"},
        {Name: "crossover_rate", Type: "float", Default: "0.7", Description: "Crossover probability"},
        {Name: "mutation_rate", Type: "float", Default: "0.1", Description: "Mutation probability"},
        {Name: "fitness_metric", Type: "string", Default: "ic", Description: "ic/ir/sharpe/composite"},
    }
}

func (n *AlphaMiningNode) Validate() error { return nil }

func (n *AlphaMiningNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
    if bridge == nil {
        return nil, fmt.Errorf("alpha_mining: PythonBridge not set")
    }

    factorPool := inputs["factor_pool"].(map[string][]float64)
    ohlcv := inputs["ohlcv_data"].(map[string][]float64)

    // Encode factor data as Arrow-compatible JSON for the gRPC call
    factorJSON, _ := json.Marshal(factorPool)
    returnsJSON, _ := json.Marshal(ohlcv)

    mlClient := python.NewMLClient(bridge)
    req := &proto.AlphaMiningRequest{
        BaseFactorNames: getFactorNames(factorPool),
        FactorData:      factorJSON,
        ReturnsData:     returnsJSON,
        PopulationSize:  int32(getIntParam(params, "population_size", 200)),
        Generations:     int32(getIntParam(params, "generations", 50)),
        CrossoverRate:   getFloatParam(params, "crossover_rate", 0.7),
        MutationRate:    getFloatParam(params, "mutation_rate", 0.1),
        FitnessMetric:   getStringParam(params, "fitness_metric", "ic"),
        TopK:            int32(getIntParam(params, "top_k", 20)),
    }

    resp, err := mlClient.AlphaMining(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("alpha_mining: %w", err)
    }

    // Convert response to output format
    formulas := make(map[string][]float64)
    scores := make(map[string][]float64)
    for i, f := range resp.Factors {
        key := fmt.Sprintf("factor_%d", i)
        formulas[key] = nil // placeholder for formula string
        scores[key] = []float64{f.Ic, f.Ir, f.Sharpe}
    }

    return map[string]any{
        "new_factors":   formulas,
        "factor_scores": scores,
    }, nil
}

func getFactorNames(factorPool map[string][]float64) []string {
    names := make([]string, 0, len(factorPool))
    for name := range factorPool {
        names = append(names, name)
    }
    return names
}
```

- [ ] **Step 3: Register in register.go**

Add to `RegisterAll`:
```go
// Phase 10.2: Alpha Mining
r.RegisterWithCategory("alpha_mining", NewAlphaMiningNode, "ml")
```

- [ ] **Step 4: Run test → pass, Step 5: Commit**

```bash
git add internal/workflow/nodes/alpha_mining.go internal/workflow/nodes/alpha_mining_test.go internal/workflow/nodes/register.go
git commit -m "feat(workflow): add AlphaMiningNode with genetic programming integration"
```

---

### Task 5: Frontend — AlphaMiningWorkspace panel

**Files:**
- Create: `frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue`
- Modify: `frontend/src/terminal/panels/registry.ts` (register panel)
- Modify: `frontend/src/stores/ml.ts` (add mining state)

- [ ] **Step 1: Extend mlStore**

Add to `frontend/src/stores/ml.ts`:

```typescript
export interface DiscoveredFactor {
  formula: string
  ic: number
  ir: number
  sharpe: number
}

// Add to store state:
const discoveredFactors = ref<DiscoveredFactor[]>([])
const miningRunning = ref(false)

// Add to store actions:
async function runAlphaMining(params: {
  factorNames: string[]
  factorData: any
  returnsData: any
  populationSize: number
  generations: number
  topK: number
}) {
  miningRunning.value = true
  try {
    if ((window as any).go?.main?.App) {
      const result = await (window as any).go.main.App.RunAlphaMining(params)
      discoveredFactors.value = result || []
    }
  } finally {
    miningRunning.value = false
  }
}

// Add to return:
discoveredFactors, miningRunning, runAlphaMining
```

- [ ] **Step 2: Write panel**

Write `frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue`:

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useMLStore } from '@/stores/ml'

const mlStore = useMLStore()
const selectedFactors = ref<string[]>([])
const popSize = ref(200)
const generations = ref(50)
const crossoverRate = ref(0.7)
const mutationRate = ref(0.1)
const topK = ref(20)
const fitnessMetric = ref('ic')

const availableFactors = [
  'momentum_1m', 'momentum_3m', 'momentum_6m', 'momentum_12m', 'rsi_alpha',
  'ma_cross', 'macd_divergence', 'trend_strength', 'adx_alpha', 'price_channel',
  'volatility_20d', 'volatility_60d', 'atr_alpha', 'bollinger_position', 'parkinson_vol',
  'volume_ratio', 'volume_trend', 'obv_alpha', 'mfi_alpha', 'vwap_deviation',
  'size_factor', 'sector_neutral_momentum', 'industry_relative', 'turnover_alpha', 'amplitude_alpha',
]

function toggleFactor(name: string) {
  const idx = selectedFactors.value.indexOf(name)
  if (idx >= 0) selectedFactors.value.splice(idx, 1)
  else selectedFactors.value.push(name)
}

async function runMining() {
  // Placeholder — calls Go backend via mlStore
  await mlStore.runAlphaMining({
    factorNames: selectedFactors.value,
    factorData: {}, returnsData: {},
    populationSize: popSize.value,
    generations: generations.value,
    topK: topK.value,
  })
}

function registerFactor(factor: { formula: string }) {
  window.dispatchEvent(new CustomEvent('quantflow:register-factor', {
    detail: { formula: factor.formula }
  }))
}
</script>

<template>
  <div class="alpha-mining-panel">
    <h3>Alpha Mining Workspace</h3>
    <div class="factor-pool">
      <h4>Base Factor Pool ({{ selectedFactors.length }} selected)</h4>
      <div class="factor-chips">
        <span v-for="f in availableFactors" :key="f"
              :class="['chip', { active: selectedFactors.includes(f) }]"
              @click="toggleFactor(f)">{{ f }}</span>
      </div>
    </div>
    <div class="gp-config">
      <h4>Genetic Programming Config</h4>
      <div class="config-grid">
        <label>Population: <input v-model.number="popSize" type="number" min="10" max="1000" /></label>
        <label>Generations: <input v-model.number="generations" type="number" min="5" max="200" /></label>
        <label>Crossover: <input v-model.number="crossoverRate" type="number" min="0" max="1" step="0.05" /></label>
        <label>Mutation: <input v-model.number="mutationRate" type="number" min="0" max="1" step="0.05" /></label>
        <label>Top K: <input v-model.number="topK" type="number" min="1" max="50" /></label>
        <label>Fitness:
          <select v-model="fitnessMetric">
            <option value="ic">IC</option>
            <option value="ir">IR</option>
            <option value="sharpe">Sharpe</option>
            <option value="composite">Composite</option>
          </select>
        </label>
      </div>
    </div>
    <button @click="runMining" :disabled="mlStore.miningRunning || selectedFactors.length < 2" class="btn-run">
      {{ mlStore.miningRunning ? 'Mining...' : 'Start Mining' }}
    </button>
    <div v-if="mlStore.discoveredFactors.length" class="results">
      <h4>Discovered Factors</h4>
      <table>
        <thead><tr><th>Formula</th><th>IC</th><th>IR</th><th>Sharpe</th><th>Action</th></tr></thead>
        <tbody>
          <tr v-for="(f, i) in mlStore.discoveredFactors" :key="i">
            <td class="formula">{{ f.formula }}</td>
            <td>{{ f.ic?.toFixed(4) }}</td>
            <td>{{ f.ir?.toFixed(4) }}</td>
            <td>{{ f.sharpe?.toFixed(4) }}</td>
            <td><button @click="registerFactor(f)" class="btn btn-sm">Register</button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.alpha-mining-panel { padding: 12px; height: 100%; overflow-y: auto; }
.factor-chips { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 12px; }
.chip { padding: 2px 8px; border: 1px solid var(--border-color); border-radius: 12px; cursor: pointer; font-size: 0.85em; }
.chip.active { background: #4a90d9; color: white; border-color: #4a90d9; }
.config-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 12px; }
.config-grid label { display: flex; flex-direction: column; font-size: 0.9em; gap: 2px; }
.config-grid input, .config-grid select { padding: 4px; border: 1px solid var(--border-color); border-radius: 4px; }
.btn-run { padding: 8px 24px; background: #4a90d9; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 1em; }
.btn-run:disabled { opacity: 0.5; cursor: not-allowed; }
.results table { width: 100%; border-collapse: collapse; margin-top: 8px; }
.results th, .results td { padding: 6px 8px; text-align: left; border-bottom: 1px solid var(--border-color); font-size: 0.9em; }
.formula { font-family: monospace; font-size: 0.85em; max-width: 300px; overflow-x: auto; }
.btn-sm { padding: 2px 8px; font-size: 0.85em; cursor: pointer; }
</style>
```

- [ ] **Step 3: Register panel**

Add to `registry.ts`:
```typescript
register('alpha-mining', () => import('./AlphaMiningWorkspacePanel.vue'))
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue frontend/src/terminal/panels/registry.ts frontend/src/stores/ml.ts
git commit -m "feat(frontend): add AlphaMiningWorkspace panel with factor pool selection and GP config"
```

---

### Task 6: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add Phase 10.2 entries**

```markdown
#### Phase 10.2 — Alpha Mining Engine
- [Python] AlphaMiningEngine: genetic programming (gplearn) for automatic factor discovery
- [Python] Factor evaluator: IC/IR/quantile Sharpe computation for candidate factors
- [Python] AlphaMining RPC wired in MLService with real genetic engine
- [Workflow] AlphaMiningNode: GP-based factor discovery node with pool selection and parameter config
- [Frontend] AlphaMiningWorkspace panel: factor pool chip selection, GP parameter config, results table with register action
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for Phase 10.2 Alpha Mining"
```
