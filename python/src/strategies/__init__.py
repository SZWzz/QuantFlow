"""量化交易策略库。

包含 16 个基于技术指标的 A 股交易策略，所有策略均继承 Strategy 基类，
实现 init() 和 next(bar) 方法。

可用策略::

    from src.strategies import available_strategies
    print(available_strategies())

    # 加载单个策略
    from src.strategies.ma_cross import MACrossStrategy
    from src.strategies.macd_cross import MACDCrossStrategy

策略列表:
    - bias_reversal     : 乖离率反转策略
    - bollinger_breakout: 布林带突破策略
    - cci_breakout      : CCI 突破策略
    - dmi_trend         : DMI 趋势策略
    - expma_cross       : EXPMA 交叉策略
    - kdj_golden        : KDJ 金叉策略
    - ma_cross          : 双均线交叉策略
    - macd_cross        : MACD 交叉策略
    - mfi_volume        : MFI 量价策略
    - mtm_momentum      : MTM 动量策略
    - obv_trend         : OBV 能量潮策略
    - rsi_reversal      : RSI 反转策略
    - trix_cross        : TRIX 交叉策略
    - turtle_breakout   : 海龟突破策略
    - volume_price      : 量价关系策略
    - zhuoyao_momentum  : 捉妖大师动量策略
"""

from .base import Strategy, crossover  # noqa: F401

__all__ = [
    "Strategy",
    "crossover",
    "available_strategies",
    "get_strategy",
]


def available_strategies() -> list[dict[str, str]]:
    """返回所有可用策略的列表。"""
    return [
        {"name": "bias_reversal", "class": "BiasReversalStrategy", "description": "乖离率反转策略"},
        {"name": "bollinger_breakout", "class": "BollingerBreakoutStrategy", "description": "布林带突破策略"},
        {"name": "cci_breakout", "class": "CCIBreakoutStrategy", "description": "CCI 突破策略"},
        {"name": "dmi_trend", "class": "DMITrendStrategy", "description": "DMI 趋势策略"},
        {"name": "expma_cross", "class": "EXPMAStrategy", "description": "EXPMA 交叉策略"},
        {"name": "kdj_golden", "class": "KDJGoldenCrossStrategy", "description": "KDJ 金叉策略"},
        {"name": "ma_cross", "class": "MACrossStrategy", "description": "双均线交叉策略"},
        {"name": "macd_cross", "class": "MACDCrossStrategy", "description": "MACD 交叉策略"},
        {"name": "mfi_volume", "class": "MFIVolumeStrategy", "description": "MFI 量价策略"},
        {"name": "mtm_momentum", "class": "MTMMomentumStrategy", "description": "MTM 动量策略"},
        {"name": "obv_trend", "class": "OBVTrendStrategy", "description": "OBV 能量潮策略"},
        {"name": "rsi_reversal", "class": "RSIReversalStrategy", "description": "RSI 反转策略"},
        {"name": "trix_cross", "class": "TRIXCrossStrategy", "description": "TRIX 交叉策略"},
        {"name": "turtle_breakout", "class": "TurtleBreakoutStrategy", "description": "海龟突破策略"},
        {"name": "volume_price", "class": "VolumePriceStrategy", "description": "量价关系策略"},
        {"name": "zhuoyao_momentum", "class": "ZhuoyaoMomentumStrategy", "description": "捉妖大师动量策略"},
    ]


def get_strategy(name: str):
    """按名称加载策略类。

    Args:
        name: 策略名称，如 "ma_cross", "macd_cross"

    Returns:
        Strategy 子类

    Raises:
        ValueError: 策略名称不存在
    """
    strategy_map = {
        "bias_reversal": "bias_reversal",
        "bollinger_breakout": "bollinger_breakout",
        "cci_breakout": "cci_breakout",
        "dmi_trend": "dmi_trend",
        "expma_cross": "expma_cross",
        "kdj_golden": "kdj_golden",
        "ma_cross": "ma_cross",
        "macd_cross": "macd_cross",
        "mfi_volume": "mfi_volume",
        "mtm_momentum": "mtm_momentum",
        "obv_trend": "obv_trend",
        "rsi_reversal": "rsi_reversal",
        "trix_cross": "trix_cross",
        "turtle_breakout": "turtle_breakout",
        "volume_price": "volume_price",
        "zhuoyao_momentum": "zhuoyao_momentum",
    }
    if name not in strategy_map:
        raise ValueError(f"未知策略: {name}。可用策略: {list(strategy_map.keys())}")

    module_name = strategy_map[name]
    import importlib
    module = importlib.import_module(f".{module_name}", package=__package__)
    return module
