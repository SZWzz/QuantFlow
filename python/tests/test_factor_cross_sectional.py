"""Tests for cross-sectional factor functions.

Covers: zscore_momentum_20d, rank_momentum_20d, zscore_volatility_20d,
        zscore_volume_ratio_5d, size_factor.
"""
import numpy as np
import pandas as pd
import pyarrow as pa
import pyarrow.ipc as ipc
import pytest

from src.factor.registry import compute


@pytest.fixture
def multi_symbol_ohlcv():
    """Synthetic multi-symbol OHLCV with 3 symbols, 60 days each."""
    np.random.seed(42)
    n_days = 60
    symbols = ["A", "B", "C"]

    frames = []
    for sym in symbols:
        # Each symbol has a different drift so their momentum differs
        drift = {"A": 0.1, "B": 0.05, "C": -0.02}[sym]
        close = 100 + np.arange(n_days) * drift + np.cumsum(np.random.randn(n_days) * 0.3)
        volume = np.random.randint(5000, 50000, n_days)
        dates = pd.date_range("2025-01-01", periods=n_days, freq="D")
        frames.append(pd.DataFrame({
            "date": dates,
            "symbol": sym,
            "open": close - np.random.randn(n_days) * 0.1,
            "high": close + np.abs(np.random.randn(n_days) * 0.2),
            "low": close - np.abs(np.random.randn(n_days) * 0.2),
            "close": close,
            "volume": volume,
        }))
    return pd.concat(frames, ignore_index=True)


@pytest.fixture
def single_symbol_ohlcv():
    """Single-symbol DataFrame (no 'symbol' column) to test fallback."""
    np.random.seed(42)
    n = 60
    close = 100 + np.cumsum(np.random.randn(n) * 0.5)
    volume = np.random.randint(5000, 50000, n)
    return pd.DataFrame({
        "open": close - np.random.randn(n) * 0.1,
        "high": close + np.abs(np.random.randn(n) * 0.3),
        "low": close - np.abs(np.random.randn(n) * 0.3),
        "close": close,
        "volume": volume,
    })


class TestZScoreMomentum20d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_momentum_20d
        result = zscore_momentum_20d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_returns_nan_for_single_symbol(self, single_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_momentum_20d
        result = zscore_momentum_20d(single_symbol_ohlcv, {})
        assert result.isna().all()

    def test_zscore_mean_near_zero_per_date(self, multi_symbol_ohlcv):
        """Z-scores should have mean ≈ 0 within each date group (ignoring NaNs)."""
        from src.factor.cross_sectional import zscore_momentum_20d
        result = zscore_momentum_20d(multi_symbol_ohlcv, {})
        df = multi_symbol_ohlcv.copy()
        df["z"] = result
        # Check a few dates after the burn-in
        for date in df["date"].unique()[25:30]:
            group = df.loc[df["date"] == date, "z"].dropna()
            if len(group) > 0:
                assert abs(group.mean()) < 1e-10


class TestRankMomentum20d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import rank_momentum_20d
        result = rank_momentum_20d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_rank_between_zero_and_one(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import rank_momentum_20d
        result = rank_momentum_20d(multi_symbol_ohlcv, {})
        valid = result.dropna()
        assert (valid >= 0).all()
        assert (valid <= 1).all()

    def test_returns_nan_for_single_symbol(self, single_symbol_ohlcv):
        from src.factor.cross_sectional import rank_momentum_20d
        result = rank_momentum_20d(single_symbol_ohlcv, {})
        assert result.isna().all()


class TestZScoreVolatility20d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volatility_20d
        result = zscore_volatility_20d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_zscore_mean_near_zero_per_date(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volatility_20d
        result = zscore_volatility_20d(multi_symbol_ohlcv, {})
        df = multi_symbol_ohlcv.copy()
        df["z"] = result
        for date in df["date"].unique()[25:30]:
            group = df.loc[df["date"] == date, "z"].dropna()
            if len(group) > 0:
                assert abs(group.mean()) < 1e-10


class TestZScoreVolumeRatio5d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volume_ratio_5d
        result = zscore_volume_ratio_5d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_zscore_mean_near_zero_per_date(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volume_ratio_5d
        result = zscore_volume_ratio_5d(multi_symbol_ohlcv, {})
        df = multi_symbol_ohlcv.copy()
        df["z"] = result
        for date in df["date"].unique()[10:15]:
            group = df.loc[df["date"] == date, "z"].dropna()
            if len(group) > 0:
                assert abs(group.mean()) < 1e-10


class TestSizeFactor:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import size_factor
        result = size_factor(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_returns_series_with_single_symbol(self, single_symbol_ohlcv):
        """size_factor works on single-symbol DataFrames too (no symbol col)."""
        from src.factor.cross_sectional import size_factor
        result = size_factor(single_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(single_symbol_ohlcv)

    def test_values_are_finite_where_available(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import size_factor
        result = size_factor(multi_symbol_ohlcv, {})
        valid = result.dropna()
        assert np.isfinite(valid).all()


# --- RPC-path regression tests (P0-3) ---
# These tests call `compute()` directly with a full multi-symbol panel to
# validate that cross-sectional factor logic is correct on a full panel.
# The bug was in engine.py filtering to a single symbol before calling the
# factor function; these tests serve as the baseline proving the factor
# math itself is correct. The engine.py fix makes the RPC path also pass
# full panels for cross-sectional factors.


def _encode_ohlcv(df: pd.DataFrame) -> bytes:
    """Encode a DataFrame to Arrow IPC stream bytes (matches gRPC wire format)."""
    table = pa.Table.from_pandas(df, preserve_index=False)
    sink = pa.BufferOutputStream()
    with ipc.new_stream(sink, table.schema) as writer:
        writer.write_table(table)
    return sink.getvalue().to_pybytes()


def _make_panel(symbols, dates, base_price=10.0):
    """Build a multi-symbol OHLCV panel with distinct momentum per symbol.

    Each symbol grows at a DIFFERENT daily rate so that 20d pct_change
    momentum is genuinely distinct across symbols (pct_change normalizes
    out the base price, so distinct bases alone would yield identical
    momentum — which would degenerate the cross-sectional zscore).
    """
    # Distinct daily growth rates → distinct 20d momentum per symbol.
    growth_rates = [1.01, 1.02, 1.005]
    rows = []
    for i, sym in enumerate(symbols):
        price = base_price * (1 + i * 0.5)  # distinct price levels
        growth = growth_rates[i % len(growth_rates)]
        for d in dates:
            rows.append({
                "symbol": sym, "date": d,
                "open": price, "high": price, "low": price,
                "close": price, "volume": 10000,
            })
            price *= growth  # upward drift, distinct per symbol
    return pd.DataFrame(rows)


def test_zscore_momentum_cross_sectional_nonzero():
    """zscore_momentum_20d on 3+ symbols must NOT be all zeros.

    With single-symbol filtering (the bug), std=0 → zscore=0 for every row.
    With full-panel computation, cross-sectional zscore should have
    non-zero variance and sum to ~0 per date.
    """
    symbols = ["AAA", "BBB", "CCC"]
    dates = pd.date_range("2026-01-01", periods=30, freq="D").astype(str)
    df = _make_panel(symbols, dates)

    # Compute on the FULL panel (what engine.py should do for cross-sectional)
    values = compute("zscore_momentum_20d", df, {})

    # Drop the warmup NaNs (first 20 days have no 20d momentum)
    valid = values.dropna()
    assert len(valid) > 0, "no valid zscore values produced"

    # BUG: with single-symbol filtering, every zscore would be 0.
    assert valid.std() > 1e-6, (
        f"cross-sectional zscore std is ~0 — factor is broken. "
        f"std={valid.std()}, values sample: {valid.head().tolist()}"
    )

    # Cross-sectional zscore property: per-date mean ≈ 0
    df_out = df.copy()
    df_out["zscore"] = values.values
    df_out = df_out.dropna(subset=["zscore"])
    per_date_mean = df_out.groupby("date")["zscore"].mean()
    assert per_date_mean.abs().max() < 1e-6, (
        f"per-date zscore mean should be ~0, got max abs {per_date_mean.abs().max()}"
    )


def test_rank_momentum_cross_sectional_distribution():
    """rank_momentum_20d on 3 symbols should produce ranks in [0,1] with spread."""
    symbols = ["AAA", "BBB", "CCC"]
    dates = pd.date_range("2026-01-01", periods=30, freq="D").astype(str)
    df = _make_panel(symbols, dates)

    values = compute("rank_momentum_20d", df, {})
    valid = values.dropna()

    # BUG: with single-symbol filtering, rank would be 0.5 for every row.
    assert valid.nunique() > 1, (
        f"rank has only 1 unique value — factor is broken. "
        f"unique={valid.nunique()}, values: {valid.unique().tolist()}"
    )
    assert valid.min() >= 0 and valid.max() <= 1
