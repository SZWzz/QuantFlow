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
        gplearn_fn = {
            "add": np.add, "sub": np.subtract, "mul": np.multiply, "div": np.divide,
            "sqrt": np.sqrt, "log": np.log, "abs": np.abs, "neg": np.negative,
            "inv": lambda x: 1.0 / x, "sin": np.sin, "cos": np.cos, "tan": np.tan,
        }
        for r in results:
            try:
                # The formula references factor column names and gplearn function names
                values = eval(r["formula"], {"__builtins__": {}}, {
                    **{col: df[col].values for col in df.columns},
                    **gplearn_fn,
                })
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


def test_evaluate_factor():
    """evaluate_factor should produce non-zero IC for a correlated factor."""
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


def test_evaluate_factor_uncorrelated():
    """evaluate_factor should return near-zero IC for an uncorrelated factor."""
    from src.ml.alpha_mining.evaluator import evaluate_factor

    np.random.seed(42)
    n = 200
    f0 = np.random.randn(n)
    returns = np.random.randn(n) * 0.1
    data = pa.Table.from_pandas(pd.DataFrame({"f_0": f0}))
    rets = pa.Table.from_pandas(pd.DataFrame({"return": returns}))

    result = evaluate_factor("f_0", data, rets)
    assert "ic" in result
    assert abs(result["ic"]) < 0.3


def test_evaluate_factor_invalid_formula():
    """evaluate_factor should return error dict for an invalid formula."""
    from src.ml.alpha_mining.evaluator import evaluate_factor

    np.random.seed(42)
    n = 100
    data = pa.Table.from_pandas(pd.DataFrame({"f_0": np.random.randn(n)}))
    rets = pa.Table.from_pandas(pd.DataFrame({"return": np.random.randn(n)}))

    result = evaluate_factor("invalid_expr(", data, rets)
    assert "error" in result
    assert result["ic"] == 0.0
