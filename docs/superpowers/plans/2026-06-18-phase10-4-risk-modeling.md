# Phase 10.4: Risk Modeling — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Add GARCH volatility modeling and covariance matrix estimation (Python) + RiskModelNode (Go workflow) + RiskDashboard extension (Vue).

**Architecture:** Python RiskEngine provides GARCH/GJR-GARCH/EGARCH volatility modeling and Ledoit-Wolf/DCC covariance estimation via Arrow IPC. Go RiskModelNode calls Python via gRPC. Results feed into existing RiskDashboard panel (extended with volatility heatmap).

**Tech Stack:** Python 3.12+ (arch, numpy, pandas, pyarrow), Go 1.22+, Vue 3.

**Depends on:** Phase 10.1 (proto, MLService, gRPC infrastructure).

## Global Constraints
- `arch` is an optional dependency — graceful degradation when not installed
- All RPCs return Arrow IPC encoded data for zero-copy transfer
- Follow existing patterns for nodes, gRPC services, and panels

---

### Task 1: Python RiskEngine

**Files:**
- Create: `python/src/ml/risk/__init__.py`
- Create: `python/src/ml/risk/garch.py`
- Create: `python/src/ml/risk/covariance.py`
- Create: `python/tests/test_risk_engine.py`

- [ ] **Step 1: Write test**

```python
import numpy as np
import pyarrow as pa
import pytest

try:
    import arch
    HAS_ARCH = True
except ImportError:
    HAS_ARCH = False


@pytest.fixture
def sample_returns():
    np.random.seed(42)
    returns = np.random.randn(500) * 0.01
    return pa.Table.from_pandas(pd.DataFrame({"return": returns}))


@pytest.mark.skipif(not HAS_ARCH, reason="arch not installed")
class TestGARCH:
    def test_garch_fit(self, sample_returns):
        from src.ml.risk.garch import GARCHEngine
        
        engine = GARCHEngine()
        result = engine.fit(sample_returns, {"model_type": "garch", "p": "1", "q": "1"})
        
        assert "volatility" in result
        assert len(result["volatility"]) > 0

    def test_gjr_garch(self, sample_returns):
        from src.ml.risk.garch import GARCHEngine
        
        engine = GARCHEngine()
        result = engine.fit(sample_returns, {"model_type": "gjr_garch", "p": "1", "q": "1"})
        assert "volatility" in result


@pytest.mark.skipif(not HAS_ARCH, reason="arch not installed")
class TestCovariance:
    def test_ledoit_wolf(self, sample_returns):
        from src.ml.risk.covariance import CovarianceEngine
        
        # Create multi-asset returns
        np.random.seed(42)
        returns_multi = np.random.randn(500, 3) * 0.01
        columns = [f"asset_{i}" for i in range(3)]
        table = pa.Table.from_pandas(pd.DataFrame(returns_multi, columns=columns))
        
        engine = CovarianceEngine()
        result = engine.estimate(table, {"method": "ledoit_wolf"})
        
        assert "covariance" in result
        cov = result["covariance"]
        assert cov.shape == (3, 3)
```

- [ ] **Step 2: Implement GARCHEngine**

```python
"""GARCH family volatility models."""
import numpy as np
import pyarrow as pa
import logging

logger = logging.getLogger(__name__)

_HAS_ARCH = False
try:
    from arch import arch_model
    _HAS_ARCH = True
except ImportError:
    pass


class GARCHEngine:
    def _check_arch(self):
        if not _HAS_ARCH:
            raise ImportError("arch is required. Install with: pip install arch")

    def fit(self, returns: pa.Table, params: dict) -> dict:
        self._check_arch()
        model_type = params.get("model_type", "garch")
        p = int(params.get("p", 1))
        q = int(params.get("q", 1))
        
        r = returns.column("return").to_numpy().astype(np.float64) * 100  # scale to %
        
        if model_type == "gjr_garch":
            am = arch_model(r, vol="Garch", p=p, o=1, q=q)
        elif model_type == "egarch":
            am = arch_model(r, vol="EGARCH", p=p, q=q)
        else:
            am = arch_model(r, vol="Garch", p=p, q=q)
        
        res = am.fit(disp="off")
        vol = res.conditional_volatility / 100  # unscale
        
        return {"volatility": vol.tolist(), "aic": float(res.aic), "bic": float(res.bic)}
```

- [ ] **Step 3: Implement CovarianceEngine**

```python
"""Covariance matrix estimation."""
import numpy as np
import pyarrow as pa
from sklearn.covariance import LedoitWolf


class CovarianceEngine:
    def estimate(self, returns: pa.Table, params: dict) -> dict:
        method = params.get("method", "ledoit_wolf")
        df = returns.to_pandas()
        R = df.values.astype(np.float64)
        
        if method == "ledoit_wolf":
            lw = LedoitWolf().fit(R)
            cov = lw.covariance_
        else:
            cov = np.cov(R, rowvar=False)
        
        return {"covariance": cov.tolist(), "method": method}
```

- [ ] **Step 4: Run tests, commit**

---

### Task 2: Wire RiskModel into MLService

**Files:**
- Modify: `python/src/ml/engine.py` — replace RiskModel stub

```python
async def RiskModel(self, request, context):
    try:
        returns = self._decode_arrow(request.returns_data)
        if request.model_type in ("garch", "gjr_garch", "egarch"):
            from src.ml.risk.garch import GARCHEngine
            engine = GARCHEngine()
            result = engine.fit(returns, {
                "model_type": request.model_type,
                "p": request.params.get("p", "1"),
                "q": request.params.get("q", "1"),
            })
        else:
            from src.ml.risk.covariance import CovarianceEngine
            engine = CovarianceEngine()
            result = engine.estimate(returns, {"method": request.params.get("method", "ledoit_wolf")})
        
        # Encode result as Arrow
        result_df = pa.Table.from_pandas(pd.DataFrame(result))
        sink = pa.BufferOutputStream()
        with pa.ipc.new_stream(sink, result_df.schema) as w:
            w.write_table(result_df)
        
        resp = ml_pb2.RiskModelResponse(result_data=sink.getvalue().to_pybytes())
        for k, v in result.items():
            if isinstance(v, (int, float)):
                resp.metrics[k] = v
        return resp
    except ImportError as e:
        logger.warning("RiskModel: %s", e)
        return ml_pb2.RiskModelResponse()
    except Exception as e:
        logger.exception("RiskModel failed")
        return ml_pb2.RiskModelResponse()
```

---

### Task 3: Go RiskModelNode

**Files:**
- Create: `internal/workflow/nodes/risk_model.go`
- Create: `internal/workflow/nodes/risk_model_test.go`
- Modify: `internal/workflow/nodes/register.go`

Follow existing node pattern. Ports: returns_data → volatility + covariance_matrix.

---

### Task 4: Frontend — extend RiskDashboard with volatility heatmap

**Files:**
- Modify: `frontend/src/terminal/panels/RiskDashboard.vue` — add GARCH volatility chart
- Modify: `frontend/src/stores/ml.ts` — add riskModelResult state

---

### Task 5: CHANGELOG

Add Phase 10.4 entries.

---

### Task 6: Update README — final Phase 10 stats

Update node count (55→59), panel count (21→23), phase badge (Phase 10 complete).
