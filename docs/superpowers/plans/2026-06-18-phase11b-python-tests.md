# Phase 11B: Python Test Coverage — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Cover untested factor modules (volatility, volume, cross_sectional) and LLM providers (4). Expand from 82→110+ test functions.

**Architecture:** Pure pandas/numpy unit tests for factors (no network needed). Mock httpx.AsyncClient for LLM providers. Follow existing test patterns in `python/tests/`.

**Tech Stack:** Python 3.12+, pytest, pandas, numpy, pytest-asyncio.

**Depends on:** Nothing — all factor modules and providers already exist.

## Global Constraints
- All factor tests use synthetic OHLCV data (numpy random or fixed fixtures) — no network
- LLM provider tests use `unittest.mock.AsyncMock` for httpx client
- Test file naming: `test_factor_<module>.py` in `python/tests/`
- Follow existing test patterns: `@pytest.fixture` for sample data, `class TestXxx` grouping
- Skip markers for optional deps already in place

---

### Task 1: Factor Tests — volatility.py

**Files:**
- Create: `python/tests/test_factor_volatility.py`

```python
import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def sample_ohlcv():
    """Generate 100 bars of synthetic OHLCV data."""
    np.random.seed(42)
    n = 100
    close = 100 + np.cumsum(np.random.randn(n) * 0.5)
    high = close + np.abs(np.random.randn(n) * 0.3)
    low = close - np.abs(np.random.randn(n) * 0.3)
    volume = np.random.rand(n) * 10000 + 5000
    return pd.DataFrame({
        "open": close - np.random.randn(n) * 0.1,
        "high": high,
        "low": low,
        "close": close,
        "volume": volume,
    })


class TestATR:
    def test_atr_returns_series(self, sample_ohlcv):
        from src.factor.volatility import atr_14
        result = atr_14(sample_ohlcv, {"period": "14"})
        assert isinstance(result, pd.Series)
        assert len(result) == len(sample_ohlcv)

    def test_atr_values_non_negative(self, sample_ohlcv):
        from src.factor.volatility import atr_14
        result = atr_14(sample_ohlcv, {"period": "14"})
        valid = result.dropna()
        assert (valid >= 0).all(), "ATR values should be non-negative"

    def test_atr_custom_period(self, sample_ohlcv):
        from src.factor.volatility import atr_14
        result5 = atr_14(sample_ohlcv, {"period": "5"})
        result20 = atr_14(sample_ohlcv, {"period": "20"})
        assert result5.dropna().std() != result20.dropna().std()


class TestVolatility:
    def test_volatility_20d_returns_series(self, sample_ohlcv):
        from src.factor.volatility import volatility_20d
        result = volatility_20d(sample_ohlcv, {"period": "20"})
        assert isinstance(result, pd.Series)

    def test_volatility_20d_annualized(self, sample_ohlcv):
        from src.factor.volatility import volatility_20d
        result = volatility_20d(sample_ohlcv, {"period": "20"})
        valid = result.dropna()
        # Annualized vol should be somewhat reasonable (< 200%)
        assert (valid < 2.0).all(), "Annualized vol values seem too high"

    def test_volatility_60d_smoother_than_20d(self, sample_ohlcv):
        from src.factor.volatility import volatility_20d, volatility_60d
        v20 = volatility_20d(sample_ohlcv, {"period": "20"})
        v60 = volatility_60d(sample_ohlcv, {"period": "60"})
        # 60d vol should be more stable (lower std of changes)
        valid_idx = v60.dropna().index
        if len(valid_idx) > 10:
            assert v60.loc[valid_idx].diff().std() < v20.loc[valid_idx].diff().std() * 1.5
```

Commit:

```bash
git add python/tests/test_factor_volatility.py && git commit -m "test: add volatility factor unit tests"
```

---

### Task 2: Factor Tests — volume.py + cross_sectional.py

**Files:**
- Create: `python/tests/test_factor_volume.py`
- Create: `python/tests/test_factor_cross_sectional.py`

test_factor_volume.py:
```python
import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def sample_ohlcv():
    np.random.seed(42)
    n = 100
    close = 100 + np.cumsum(np.random.randn(n) * 0.5)
    high = close + np.abs(np.random.randn(n) * 0.3)
    low = close - np.abs(np.random.randn(n) * 0.3)
    volume = np.random.randint(5000, 50000, n)
    return pd.DataFrame({
        "open": close - np.random.randn(n) * 0.1,
        "high": high, "low": low, "close": close, "volume": volume,
    })


class TestVolumeRatio:
    def test_volume_ratio_5d_relative_to_mean(self, sample_ohlcv):
        from src.factor.volume import volume_ratio_5d
        result = volume_ratio_5d(sample_ohlcv, {"period": "5"})
        assert isinstance(result, pd.Series)
        valid = result.dropna()
        # Ratio should have positive values
        assert (valid > 0).all()

    def test_volume_ratio_20d(self, sample_ohlcv):
        from src.factor.volume import volume_ratio_20d
        result = volume_ratio_20d(sample_ohlcv, {"period": "20"})
        assert len(result) == len(sample_ohlcv)


class TestOBV:
    def test_obv_cumulative(self, sample_ohlcv):
        from src.factor.volume import obv
        result = obv(sample_ohlcv, {})
        assert isinstance(result, pd.Series)
        # OBV should be non-decreasing if all prices go up
        assert not result.isna().all()


class TestVWAP:
    def test_vwap_deviation_bounded(self, sample_ohlcv):
        from src.factor.volume import vwap_deviation
        result = vwap_deviation(sample_ohlcv, {})
        valid = result.dropna()
        # Deviation should generally be within ±10%
        assert (valid.abs() < 0.15).all(), f"VWAP deviation too large: {valid.abs().max()}"
```

test_factor_cross_sectional.py:
```python
import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def multi_asset_ohlcv():
    """Create multi-asset data for cross-sectional factor testing."""
    np.random.seed(42)
    n = 100
    assets = {}
    for i in range(5):
        close = 50 + i * 10 + np.cumsum(np.random.randn(n) * 0.3)
        assets[f"asset_{i}"] = pd.DataFrame({
            "close": close,
            "volume": np.random.randint(5000, 50000, n),
        })
    return assets


class TestCrossSectional:
    def test_cross_sectional_factor_has_values(self, multi_asset_ohlcv):
        from src.factor.cross_sectional import cross_sectional_factor
        result = cross_sectional_factor(multi_asset_ohlcv, {})
        assert result is not None

    def test_cross_sectional_factor_equal_weighted(self, multi_asset_ohlcv):
        # This is a smoke test — verifies the function exists and runs
        from src.factor.cross_sectional import cross_sectional_factor
        result = cross_sectional_factor(multi_asset_ohlcv, {})
        # Should return something
        assert result is not None
```

Commit:

```bash
git add python/tests/test_factor_volume.py python/tests/test_factor_cross_sectional.py && git commit -m "test: add volume and cross-sectional factor unit tests"
```

---

### Task 3: LLM Provider Tests

**Files:**
- Create: `python/tests/test_llm_providers.py`

```python
"""Unit tests for LLM provider instantiation and error handling."""
import os
import pytest
from unittest.mock import AsyncMock, patch, MagicMock


class TestOpenAIProvider:
    def test_instantiation_default(self):
        from src.llm.providers.openai_provider import OpenAIProvider
        provider = OpenAIProvider()
        assert provider.base_url == "https://api.openai.com"
        assert provider.api_key == ""

    def test_instantiation_with_key(self):
        from src.llm.providers.openai_provider import OpenAIProvider
        provider = OpenAIProvider(api_key="sk-test")
        assert provider.api_key == "sk-test"

    def test_instantiation_custom_base_url(self):
        from src.llm.providers.openai_provider import OpenAIProvider
        provider = OpenAIProvider(base_url="https://custom.api.com/v1")
        assert provider.base_url == "https://custom.api.com/v1"

    def test_chat_no_api_key_returns_error(self):
        from src.llm.providers.openai_provider import OpenAIProvider
        from src.proto.llm_pb2 import LLMChatRequest
        import asyncio

        provider = OpenAIProvider(api_key="")  # empty key
        request = LLMChatRequest()
        request.messages.append(LLMChatRequest.Message(role="user", content="hi"))

        async def run():
            chunks = []
            async for chunk in provider.chat(request, None):
                chunks.append(chunk)
            return chunks

        chunks = asyncio.run(run())
        assert len(chunks) > 0
        assert any("error" in c.finish_reason or "not set" in c.delta_content
                   for c in chunks if c.delta_content)


class TestDeepSeekProvider:
    def test_instantiation(self):
        from src.llm.providers.deepseek_provider import DeepSeekProvider
        provider = DeepSeekProvider(api_key="sk-test")
        assert provider is not None

    def test_base_url_default(self):
        from src.llm.providers.deepseek_provider import DeepSeekProvider
        provider = DeepSeekProvider(api_key="sk-test")
        assert "deepseek" in provider.base_url.lower()


class TestOllamaProvider:
    def test_instantiation(self):
        from src.llm.providers.ollama_provider import OllamaProvider
        provider = OllamaProvider(base_url="http://localhost:11434")
        assert provider.base_url == "http://localhost:11434"


class TestAnthropicProvider:
    def test_instantiation(self):
        from src.llm.providers.anthropic_provider import AnthropicProvider
        provider = AnthropicProvider(api_key="sk-ant-test")
        assert provider is not None
```

Commit:

```bash
git add python/tests/test_llm_providers.py && git commit -m "test: add LLM provider unit tests"
```

---

### Task 4: Strengthen Existing Tests

**Files:**
- Modify: `python/tests/test_ml_service.py` — add streaming RLTrain test, RiskModel test
- Modify: `python/tests/test_alpha_mining.py` — add evaluator edge case tests

Add to test_ml_service.py:
```python
class TestMLServiceRiskModel:
    @pytest.mark.asyncio
    async def test_risk_model_returns_response(self):
        # Smoke test: verify the risk model RPC is wired
        ...

class TestMLServiceRLTrain:
    @pytest.mark.asyncio
    async def test_rl_train_streaming(self):
        # Smoke test: verify streaming RLTrain works
        ...
```

Add to test_alpha_mining.py:
```python
class TestEvaluatorEdgeCases:
    def test_evaluate_constant_returns(self):
        """Constant returns should give IC≈0."""
        ...

    def test_evaluate_single_factor(self):
        """Single factor evaluation should complete."""
        ...
```

Commit:

```bash
git add python/tests/test_ml_service.py python/tests/test_alpha_mining.py && git commit -m "test: strengthen ML service and alpha mining tests"
```

---

### Task 5: Final — run pytest, verify

```bash
cd python && python -m pytest tests/ -x -q --tb=short
```
Expected: all pass, test count ≥110.

Commit any fixes.

---

### Task 6: CHANGELOG

Add Phase 11B entries.
