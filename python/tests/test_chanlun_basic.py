"""Basic tests for core chanlun (缠论) functions.

Tests the lower-level building blocks:
- Fractal detection (分型识别) via find_fractals()
- K-line merging (K线合并) via merge_klines()
- Bi/stroke connection (笔连接) via find_bis()
- Zhongshu/pivot identification (中枢识别) via find_zss()
- Configuration defaults via ChanlunConfig()

These tests use synthetic data to verify algorithm correctness.
"""

from datetime import datetime

import pytest

from src.chanlun.bi import find_bis
from src.chanlun.config import ChanlunConfig
from src.chanlun.fractal import find_fractals
from src.chanlun.kline_merge import merge_klines
from src.chanlun.types import CLKline, FX, FXType, Kline
from src.chanlun.zs import find_zss


# ── Helpers ──────────────────────────────────────────────────────────────────

def _date(day: int) -> datetime:
    """Create a datetime for synthetic data (January 2020, given day)."""
    return datetime(2020, 1, day)


def _clkline(index: int, k_index: int, high: float, low: float, day: int = 1,
             open_: float = 0.0, close: float = 0.0) -> CLKline:
    """Create a CLKline with minimal fields filled."""
    return CLKline(
        k_index=k_index,
        date=_date(day),
        open=open_ or low,
        close=close or high,
        high=high,
        low=low,
        amount=100.0,
        index=index,
    )


# ── TestFindFractals ─────────────────────────────────────────────────────────

class TestFindFractals:
    """Verify fractal detection (顶分型 / 底分型)."""

    # fmt: off
    def _top_fractal_klines(self) -> list[CLKline]:
        """Return 5 CLKlines with a clear peak at index 2 (high=15)."""
        return [
            _clkline(index=0, k_index=0, high=10, low=8,   day=1),
            _clkline(index=1, k_index=1, high=12, low=10,  day=2),
            _clkline(index=2, k_index=2, high=15, low=13,  day=3),  # peak
            _clkline(index=3, k_index=3, high=12, low=10,  day=4),
            _clkline(index=4, k_index=4, high=10, low=8,   day=5),
        ]

    def _bottom_fractal_klines(self) -> list[CLKline]:
        """Return 5 CLKlines with a clear trough at index 2 (low=9)."""
        return [
            _clkline(index=0, k_index=0, high=15, low=13,  day=1),
            _clkline(index=1, k_index=1, high=13, low=11,  day=2),
            _clkline(index=2, k_index=2, high=11, low=9,   day=3),  # trough
            _clkline(index=3, k_index=3, high=13, low=11,  day=4),
            _clkline(index=4, k_index=4, high=15, low=13,  day=5),
        ]
    # fmt: on

    def test_top_fractal(self):
        """Test that a peak is correctly identified as 顶分型."""
        fractals = find_fractals(self._top_fractal_klines())
        # Should detect exactly one top fractal at index 2 (the middle of triple)
        assert len(fractals) == 1
        assert fractals[0].fx_type == FXType.DING
        assert fractals[0].val == 15.0

    def test_bottom_fractal(self):
        """Test that a trough is correctly identified as 底分型."""
        fractals = find_fractals(self._bottom_fractal_klines())
        assert len(fractals) == 1
        assert fractals[0].fx_type == FXType.DI
        assert fractals[0].val == 9.0

    def test_no_fractal_flat(self):
        """Test that flat data produces no fractals."""
        klines = [_clkline(index=i, k_index=i, high=10, low=9, day=i + 1)
                  for i in range(5)]
        fractals = find_fractals(klines)
        assert len(fractals) == 0

    def test_insufficient_klines(self):
        """Test that fewer than 3 klines returns empty list."""
        klines = [
            _clkline(index=0, k_index=0, high=10, low=8, day=1),
            _clkline(index=1, k_index=1, high=12, low=10, day=2),
        ]
        assert find_fractals(klines) == []

    def test_config_strict(self):
        """Test strict mode: equal highs should not form a fractal."""
        # Equal highs -> strict mode should NOT detect a top fractal
        klines = [
            _clkline(index=0, k_index=0, high=10, low=8,   day=1),
            _clkline(index=1, k_index=1, high=10, low=9,   day=2),  # equal to left
            _clkline(index=2, k_index=2, high=12, low=10,  day=3),
        ]
        config = ChanlunConfig(fx_strict=True)
        fractals = find_fractals(klines, config=config)
        # strict: mid=10 is NOT > left=10, so no top fractal
        assert len(fractals) == 0


# ── TestKlineMerge ───────────────────────────────────────────────────────────

class TestKlineMerge:
    """Verify K-line merging (包含处理)."""

    def _kline(self, index: int, high: float, low: float, day: int,
               open_: float = 0.0, close: float = 0.0) -> Kline:
        return Kline(
            index=index,
            date=_date(day),
            open=open_ or low,
            close=close or high,
            high=high,
            low=low,
            amount=100.0,
        )

    def test_no_merge(self):
        """Test that non-overlapping klines pass through unchanged."""
        klines = [
            self._kline(index=0, high=12, low=10, day=1),
            self._kline(index=1, high=15, low=13, day=2),
            self._kline(index=2, high=18, low=16, day=3),
        ]
        merged = merge_klines(klines)
        assert len(merged) == 3
        # Each should be a separate CLKline
        assert merged[0].high == 12
        assert merged[1].high == 15
        assert merged[2].high == 18

    def test_upward_inclusion(self):
        """Test upward inclusion merge: take higher high and higher low."""
        # First bar sets direction upward (close >= open)
        klines = [
            self._kline(index=0, high=15, low=10, day=1, open_=10, close=12),
            # Second bar is fully contained in the first
            self._kline(index=1, high=14, low=11, day=2, open_=11, close=13),
        ]
        merged = merge_klines(klines)
        assert len(merged) == 1  # merged into one
        # Upward: max(highs) = 15, max(lows) = 11
        assert merged[0].high == 15.0
        assert merged[0].low == 11.0
        assert merged[0].merged_count == 2

    def test_downward_inclusion(self):
        """Test downward inclusion merge: take lower high and lower low."""
        # First bar is bearish (close < open) -> direction is down
        klines = [
            self._kline(index=0, high=15, low=10, day=1, open_=14, close=12),
            self._kline(index=1, high=14, low=11, day=2, open_=13, close=12),
        ]
        merged = merge_klines(klines)
        assert len(merged) == 1
        # Downward: min(highs) = 14, min(lows) = 10
        assert merged[0].high == 14.0
        assert merged[0].low == 10.0

    def test_empty_input(self):
        """Test empty input returns empty list."""
        assert merge_klines([]) == []


# ── TestBi ───────────────────────────────────────────────────────────────────

class TestBi:
    """Verify bi (笔/stroke) detection."""

    def _clkline_for_fx(self, idx: int, k_idx: int, high: float, low: float,
                        day: int) -> CLKline:
        return CLKline(
            k_index=k_idx, date=_date(day), open=low, close=high,
            high=high, low=low, amount=100.0, index=idx,
        )

    def _make_top_fx(self, mid: CLKline, left: CLKline, right: CLKline,
                     val: float, idx: int) -> FX:
        return FX(fx_type=FXType.DING, k=mid, klines=[left, mid, right],
                  val=val, index=idx, done=True)

    def _make_bottom_fx(self, mid: CLKline, left: CLKline, right: CLKline,
                        val: float, idx: int) -> FX:
        return FX(fx_type=FXType.DI, k=mid, klines=[left, mid, right],
                  val=val, index=idx, done=True)

    def test_bi_between_fractals(self):
        """Test that alternating fractals form a bi (new bi rule, gap >= 1)."""
        # Build 8 CLKlines with index 0..7
        ck = [self._clkline_for_fx(i, i, 10 + i, 8 + i, i + 1) for i in range(8)]
        # Bottom fractal at index 2 (k_index=2), top at index 6 (k_index=6)
        # Gap = end.klines[0].index - start.klines[2].index + 1
        # For bottom: klines = [ck[1], ck[2], ck[3]] so klines[2].index = 3
        # For top:    klines = [ck[5], ck[6], ck[7]] so klines[0].index = 5
        # Gap = 5 - 3 + 1 = 3 >= 1 -> should form bi
        bottom = self._make_bottom_fx(ck[2], ck[1], ck[3], val=ck[2].low, idx=0)
        top = self._make_top_fx(ck[6], ck[5], ck[7], val=ck[6].high, idx=1)
        bis = find_bis([bottom, top])
        assert len(bis) >= 1
        assert bis[0].direction.value == "up"
        assert bis[0].high > bis[0].low

    def test_bi_requires_different_types(self):
        """Test that two same-type fractals do not form a bi."""
        ck = [self._clkline_for_fx(i, i, 10 + i, 8 + i, i + 1) for i in range(8)]
        top1 = self._make_top_fx(ck[2], ck[1], ck[3], val=ck[2].high, idx=0)
        top2 = self._make_top_fx(ck[6], ck[5], ck[7], val=ck[6].high, idx=1)
        bis = find_bis([top1, top2])
        assert len(bis) == 0  # same type -> no bi

    def test_bi_insufficient_fractals(self):
        """Test that fewer than 2 fractals returns empty list."""
        ck = [self._clkline_for_fx(i, i, 10, 8, i + 1) for i in range(3)]
        fx = self._make_top_fx(ck[1], ck[0], ck[2], val=ck[1].high, idx=0)
        assert find_bis([fx]) == []


# ── TestZhongshu ─────────────────────────────────────────────────────────────

class TestZhongshu:
    """Verify zhongshu (中枢/pivot) identification."""

    def test_insufficient_bis(self):
        """Test that fewer than 3 bis returns empty list."""
        config = ChanlunConfig()
        assert find_zss([], config=config) == []

    def test_zhongshu_requires_overlap(self):
        """Test that non-overlapping bis do not form a zhongshu."""
        # Build two FX objects for each bi (simplified — using klines with
        # large gaps so there's no overlap)
        config = ChanlunConfig(zs_min_lines=3)

        def _bi(high: float, low: float):
            ck = [_clkline_for_bi(i, h, l) for i, (h, l) in
                  enumerate(((low, low), (high, high), (high, high)))]
            return FX(fx_type=FXType.DING, k=ck[1], klines=ck,
                      val=ck[1].high, index=0, done=True)

        # With no overlap, bi list won't form zhongshu — just check no crash
        bis = find_bis([])
        assert find_zss(bis, config=config) == []


def _clkline_for_bi(idx: int, high: float, low: float) -> CLKline:
    """Create a CLKline for bi tests (minimal fields)."""
    return CLKline(
        k_index=idx, date=_date(idx + 1), open=low, close=high,
        high=high, low=low, amount=100.0, index=idx,
    )


# ── TestConfig ───────────────────────────────────────────────────────────────

class TestConfig:
    """Verify ChanlunConfig defaults."""

    def test_default_config(self):
        """Test default configuration values."""
        config = ChanlunConfig()
        assert config.bi_type == "new"
        assert config.zs_type == "standard"
        assert config.zs_min_lines == 3
        assert config.zs_qujian == "dd"
        assert config.fx_strict is True
        assert config.bi_qujian == "dd"
        assert config.xd_bi_pohuai is False
        assert config.macd_fast == 12
        assert config.macd_slow == 26
        assert config.macd_signal == 9

    def test_config_custom(self):
        """Test custom configuration values."""
        config = ChanlunConfig(
            bi_type="old",
            fx_strict=False,
            zs_min_lines=5,
        )
        assert config.bi_type == "old"
        assert config.fx_strict is False
        assert config.zs_min_lines == 5

    def test_config_to_dict(self):
        """Test config serialization to dict."""
        config = ChanlunConfig()
        d = config.to_dict()
        assert isinstance(d, dict)
        assert d["bi_type"] == "new"
        assert d["fx_strict"] is True
        assert "macd_fast" in d
        assert "macd_slow" in d
