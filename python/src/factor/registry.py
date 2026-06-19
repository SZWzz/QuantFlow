"""Factor registry — maps factor names to implementations with metadata."""
from dataclasses import dataclass, field
from typing import Callable, Dict, Any, List
import pandas as pd


@dataclass
class FactorMeta:
    name: str
    category: str
    description: str
    default_params: Dict[str, str] = field(default_factory=dict)


# Global registries
_registry: Dict[str, FactorMeta] = {}
_compute_funcs: Dict[str, Callable] = {}


def register(meta: FactorMeta):
    """Decorator to register a factor computation function.

    Usage:
        @register(FactorMeta(name="my_factor", category="momentum", description="..."))
        def my_factor(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
            ...
    """

    def decorator(func: Callable):
        _registry[meta.name] = meta
        _compute_funcs[meta.name] = func
        return func

    return decorator


def list_factors() -> List[FactorMeta]:
    """Return all registered factor metadata."""
    return list(_registry.values())


def compute(factor_name: str, ohlcv: pd.DataFrame, params: Dict[str, Any]) -> pd.Series:
    """Compute a factor for the given OHLCV data.

    Args:
        factor_name: Name of the factor to compute (e.g., "momentum_20d").
        ohlcv: DataFrame with columns 'open', 'high', 'low', 'close', 'volume'.
        params: Factor-specific parameters (merged with defaults).

    Returns:
        Series of factor values with the same index as ohlcv.

    Raises:
        KeyError: If factor_name is not registered.
    """
    if factor_name not in _compute_funcs:
        raise KeyError(f"Unknown factor: {factor_name}")

    meta = _registry[factor_name]
    # Merge default params with provided params
    merged = {**meta.default_params, **params}
    return _compute_funcs[factor_name](ohlcv, merged)
