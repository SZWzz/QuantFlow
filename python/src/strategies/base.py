"""策略基类和辅助函数。

提供策略编写的声明式 API：
- Strategy: 策略基类，init() 注册指标，next() 生成信号
- crossover: 金叉检测辅助函数
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any

import numpy as np
import numpy.typing as npt
import pandas as pd

if True:
    from collections.abc import Callable

    NDArray = npt.NDArray[np.float64]
else:
    NDArray = np.ndarray


# ── 金叉检测 ─────────────────────────────────────────────────────────────────────


def crossover(
    a: NDArray | pd.Series,
    b: NDArray | pd.Series,
) -> NDArray:
    """检测 a 从下方穿越 b（金叉）。

    Args:
        a: 快线序列
        b: 慢线序列

    Returns:
        bool 数组，True 表示发生金叉
    """
    if isinstance(a, pd.Series):
        a = a.to_numpy().astype(np.float64)
    if isinstance(b, pd.Series):
        b = b.to_numpy().astype(np.float64)

    a = np.asarray(a, dtype=np.float64)
    b = np.asarray(b, dtype=np.float64)

    mask = np.zeros(len(a), dtype=bool)
    if len(a) > 1:
        mask[1:] = (a[:-1] <= b[:-1]) & (a[1:] > b[1:])
    return mask


# ── 策略基类 ───────────────────────────────────────────────────────────────────


class Strategy(ABC):
    """策略基类，提供声明式回测 API。

    用户子类实现：
        - init(): 注册指标（通过 self.I()）
        - next(): 生成交易信号（通过 self.buy()/self.sell()）

    内部状态：
        - self.data: StrategyDataProxy，访问 K线数据
        - self.position: {"size": float}，当前持仓

    示例:
        >>> class MyStrategy(Strategy):
        ...     def init(self):
        ...         self.ma5 = self.I(MyTT.MA, self.data.close, 5)
        ...         self.ma20 = self.I(MyTT.MA, self.data.close, 20)
        ...         self.cross = crossover(self.ma5, self.ma20)
        ...
        ...     def next(self):
        ...         if self.cross[self._bar_index]:
        ...             self.buy(size=100)
        ...         elif self.position["size"] > 0:
        ...             self.sell(size=0)
    """

    def __init__(self) -> None:
        self._data_proxy: StrategyDataProxy | None = None
        self._bar_index = 0
        self._signals: list[dict] = []
        self._indicators: dict[str, NDArray] = {}
        self._position_size = 0.0
        self._cash = 0.0
        self._datetime_array: NDArray | None = None

    @abstractmethod
    def init(self) -> None:
        """注册指标。在回测开始前调用一次。"""
        ...

    @abstractmethod
    def next(self) -> None:
        """生成交易信号。每根 bar 调用一次。"""
        ...

    def I(
        self, func: Callable[..., NDArray], *args: Any, **kwargs: Any
    ) -> NDArray:
        """注册指标。自动解包 _SeriesAccessor 参数。"""
        unpacked_args = []
        for arg in args:
            if isinstance(arg, _SeriesAccessor):
                unpacked_args.append(arg.raw)
            else:
                unpacked_args.append(arg)
        result = func(*unpacked_args, **kwargs)
        return result

    def buy(self, size: float = 0, price: float | None = None) -> None:
        """生成买入信号。size=0 表示全仓。"""
        if self._datetime_array is None:
            raise RuntimeError("数据未正确初始化")
        dt = int(self._datetime_array[self._bar_index]) if self._datetime_array is not None else 0
        self._signals.append({
            "datetime": dt,
            "direction": "BUY",
            "size": size,
            "price": price,
        })

    def sell(self, size: float = 0, price: float | None = None) -> None:
        """生成卖出信号。size=0 表示全仓。"""
        if self._datetime_array is None:
            raise RuntimeError("数据未正确初始化")
        dt = int(self._datetime_array[self._bar_index]) if self._datetime_array is not None else 0
        self._signals.append({
            "datetime": dt,
            "direction": "SELL",
            "size": size,
            "price": price,
        })

    @property
    def data(self) -> "StrategyDataProxy":
        """K线数据代理。"""
        if self._data_proxy is None:
            raise RuntimeError("策略未绑定数据")
        return self._data_proxy

    @property
    def position(self) -> dict[str, float]:
        """当前持仓。{"size": float}，正=多头，负=空头，0=空仓。"""
        return {"size": self._position_size}

    def _bind_data(self, df: pd.DataFrame) -> None:
        """绑定 K线数据（引擎调用）。"""
        self._data_proxy = StrategyDataProxy(df)
        if "datetime" in df.columns:
            self._datetime_array = df["datetime"].to_numpy().astype(np.float64)
        else:
            raise ValueError("DataFrame 必须包含 datetime 列")

    def _call_init(self) -> None:
        self.init()

    def _set_bar_index(self, idx: int) -> None:
        self._bar_index = idx
        if self._data_proxy is not None:
            self._data_proxy._set_index(idx)

    def _call_next(self) -> None:
        self.next()

    def _clear_signals(self) -> list[dict]:
        signals = self._signals
        self._signals = []
        return signals


# ── 数据序列访问器 ─────────────────────────────────────────────────────────────


class _SeriesAccessor:
    """数据序列访问器，支持相对索引和 numpy 数组转换。"""

    __slots__ = ("_series", "_bar_index")

    def __init__(self, series: NDArray, bar_index: int) -> None:
        self._series = series
        self._bar_index = bar_index

    def __getitem__(self, key: int) -> float:
        idx = self._bar_index + key
        if idx < 0:
            raise IndexError(f"索引 {key} 超出范围")
        return float(self._series[idx])

    def __len__(self) -> int:
        return len(self._series)

    @property
    def raw(self) -> NDArray:
        return self._series


# ── K线数据代理 ────────────────────────────────────────────────────────────────


class StrategyDataProxy:
    """K线数据代理，将 DataFrame 转为高效的 numpy 数组访问。"""

    __slots__ = ("_arrays", "_bar_index")

    def __init__(self, df: pd.DataFrame) -> None:
        self._arrays: dict[str, NDArray] = {}
        self._bar_index = 0
        for col in df.columns:
            if col == "datetime":
                continue
            arr = df[col].to_numpy()
            if len(arr) > 0 and isinstance(arr[0], np.datetime64 | pd.Timestamp):
                pass  # skip
            else:
                self._arrays[col] = arr.astype(np.float64)

    def _set_index(self, idx: int) -> None:
        self._bar_index = idx

    @property
    def open(self) -> _SeriesAccessor:
        return _SeriesAccessor(self._arrays["open"], self._bar_index)

    @property
    def close(self) -> _SeriesAccessor:
        return _SeriesAccessor(self._arrays["close"], self._bar_index)

    @property
    def high(self) -> _SeriesAccessor:
        return _SeriesAccessor(self._arrays["high"], self._bar_index)

    @property
    def low(self) -> _SeriesAccessor:
        return _SeriesAccessor(self._arrays["low"], self._bar_index)

    @property
    def vol(self) -> _SeriesAccessor:
        return _SeriesAccessor(self._arrays["vol"], self._bar_index)

    @property
    def amount(self) -> _SeriesAccessor:
        return _SeriesAccessor(self._arrays["amount"], self._bar_index)

    def __getattr__(self, name: str) -> _SeriesAccessor:
        if name not in self._arrays:
            raise AttributeError(f"列 '{name}' 不存在于数据中")
        return _SeriesAccessor(self._arrays[name], self._bar_index)
