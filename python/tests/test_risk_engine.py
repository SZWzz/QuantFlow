"""Tests for Risk Modeling Engine (Phase 10.4)."""
import numpy as np
import pyarrow as pa
import pytest

try:
    import pandas as pd
    HAS_PANDAS = True
except ImportError:
    HAS_PANDAS = False

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


@pytest.fixture
def sample_multi_returns():
    np.random.seed(42)
    returns_multi = np.random.randn(500, 3) * 0.01
    columns = [f"asset_{i}" for i in range(3)]
    return pa.Table.from_pandas(pd.DataFrame(returns_multi, columns=columns))


@pytest.mark.skipif(not HAS_ARCH, reason="arch not installed")
class TestGARCH:
    def test_garch_fit(self, sample_returns):
        from src.ml.risk.garch import GARCHEngine

        engine = GARCHEngine()
        result = engine.fit(sample_returns, {"model_type": "garch", "p": "1", "q": "1"})

        assert "volatility" in result
        assert "aic" in result
        assert "bic" in result
        assert len(result["volatility"]) > 0
        assert isinstance(result["aic"], float)
        assert isinstance(result["bic"], float)

    def test_gjr_garch_fit(self, sample_returns):
        from src.ml.risk.garch import GARCHEngine

        engine = GARCHEngine()
        result = engine.fit(sample_returns, {"model_type": "gjr_garch", "p": "1", "q": "1"})

        assert "volatility" in result
        assert len(result["volatility"]) > 0

    def test_egarch_fit(self, sample_returns):
        from src.ml.risk.garch import GARCHEngine

        engine = GARCHEngine()
        result = engine.fit(sample_returns, {"model_type": "egarch", "p": "1", "q": "1"})

        assert "volatility" in result
        assert len(result["volatility"]) > 0

    def test_volatility_non_negative(self, sample_returns):
        from src.ml.risk.garch import GARCHEngine

        engine = GARCHEngine()
        result = engine.fit(sample_returns, {"model_type": "garch", "p": "1", "q": "1"})

        for v in result["volatility"]:
            assert v >= 0, f"volatility should be non-negative, got {v}"


@pytest.mark.skipif(not HAS_ARCH, reason="arch not installed")
class TestGARCHMissingDependency:
    def test_missing_arch_raises(self, tmp_path, monkeypatch):
        """Test that GARCHEngine raises ImportError when arch is not available."""
        from src.ml.risk.garch import GARCHEngine, _HAS_ARCH as orig_has_arch
        import src.ml.risk.garch as garch_mod

        try:
            monkeypatch.setattr(garch_mod, "_HAS_ARCH", False)
            engine = GARCHEngine()
            with pytest.raises(ImportError, match="arch"):
                engine.fit(None, {"model_type": "garch"})
        finally:
            garch_mod._HAS_ARCH = orig_has_arch


class TestCovariance:
    def test_sample_covariance(self, sample_multi_returns):
        from src.ml.risk.covariance import CovarianceEngine

        engine = CovarianceEngine()
        result = engine.estimate(sample_multi_returns, {"method": "sample"})

        assert "covariance" in result
        cov = np.array(result["covariance"])
        assert cov.shape == (3, 3)
        assert result["method"] == "sample"

    def test_ledoit_wolf(self, sample_multi_returns):
        from src.ml.risk.covariance import CovarianceEngine

        engine = CovarianceEngine()
        result = engine.estimate(sample_multi_returns, {"method": "ledoit_wolf"})

        assert "covariance" in result
        cov = np.array(result["covariance"])
        assert cov.shape == (3, 3)
        assert result["method"] == "ledoit_wolf"

    def test_covariance_symmetric(self, sample_multi_returns):
        from src.ml.risk.covariance import CovarianceEngine

        engine = CovarianceEngine()
        for method in ["sample", "ledoit_wolf"]:
            result = engine.estimate(sample_multi_returns, {"method": method})
            cov = np.array(result["covariance"])
            assert np.allclose(cov, cov.T), f"covariance matrix should be symmetric for {method}"

    def test_covariance_diagonal_positive(self, sample_multi_returns):
        from src.ml.risk.covariance import CovarianceEngine

        engine = CovarianceEngine()
        for method in ["sample", "ledoit_wolf"]:
            result = engine.estimate(sample_multi_returns, {"method": method})
            cov = np.array(result["covariance"])
            for i in range(cov.shape[0]):
                assert cov[i, i] > 0, f"diagonal element should be positive for {method}"
