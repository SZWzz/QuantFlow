"""缠论（ChanLun）技术分析模块。

基于缠论理论实现 K 线合并、分型识别、笔/线段/中枢/买卖点/背驰计算。

核心 API::

    from src.chanlun import analyze, ChanlunAnalyser, ChanlunConfig

    result = analyze(df, multi_level="1D")
    print(result)

    analyser = ChanlunAnalyser("SZ000001", "DAILY")
    result = analyser.process_klines(df)
"""

from __future__ import annotations

import pandas as pd

from .analyser import ChanlunAnalyser, ChanlunResult  # noqa: F401
from .config import ChanlunConfig  # noqa: F401
from .types import (  # noqa: F401
    BC,
    BI,
    FX,
    MMD,
    XD,
    ZS,
    BCType,
    CLKline,
    Direction,
    FXType,
    Kline,
    MMDType,
)

__all__ = [
    "analyze",
    "ChanlunAnalyser",
    "ChanlunConfig",
    "ChanlunResult",
    "BC",
    "BCType",
    "BI",
    "CLKline",
    "Direction",
    "FX",
    "FXType",
    "Kline",
    "MMD",
    "MMDType",
    "XD",
    "ZS",
]


def analyze(df: pd.DataFrame, multi_level: str = None) -> dict:
    """对 K 线 DataFrame 执行缠论分析。

    Args:
        df: K 线数据，需包含 datetime, open, close, high, low, vol 列。
        multi_level: 多级别分析参数，如 "1D", "60MIN" 等。None 表示单级别分析。

    Returns:
        dict 包含:
            - fractals: 分型列表
            - bi_list: 笔列表
            - zs_list: 中枢列表
            - xd_list: 线段列表
            - mmd_list: 买卖点列表
            - beichi_list: 背驰列表
            - macd: MACD 数据
    """
    if multi_level:
        from .multi_level import analyze_multi_level
        return analyze_multi_level(df)

    analyser = ChanlunAnalyser()
    result = analyser.process_klines(df)
    d = result.to_dict()

    return {
        "fractals": result.fractals,
        "bi_list": d.get("bis", []),
        "zs_list": d.get("zss", []),
        "xd_list": d.get("xds", []),
        "mmd_list": d.get("mmds", []),
        "beichi_list": d.get("bcs", []),
        "macd": d,
    }
