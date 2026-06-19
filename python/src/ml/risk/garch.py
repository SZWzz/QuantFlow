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
    """GARCH/GJR-GARCH/EGARCH volatility modeling.

    Fits GARCH-family models to return series and returns conditional
    volatility, AIC, and BIC.
    """

    def _check_arch(self):
        if not _HAS_ARCH:
            raise ImportError("arch is required. Install with: pip install arch")

    def fit(self, returns: pa.Table, params: dict) -> dict:
        """Fit a GARCH-family model to the returns data.

        Args:
            returns: Arrow Table with 'return' column (daily log returns).
            params: dict with keys: model_type, p, q.

        Returns:
            dict with keys: volatility (list), aic (float), bic (float).
        """
        self._check_arch()
        model_type = params.get("model_type", "garch")
        p = int(params.get("p", 1))
        q = int(params.get("q", 1))

        r = returns.column("return").to_numpy().astype(np.float64) * 100  # scale to % for numerical stability

        if model_type == "gjr_garch":
            am = arch_model(r, vol="Garch", p=p, o=1, q=q)
        elif model_type == "egarch":
            am = arch_model(r, vol="EGARCH", p=p, q=q)
        else:
            am = arch_model(r, vol="Garch", p=p, q=q)

        res = am.fit(disp="off")
        vol = res.conditional_volatility / 100  # unscale back

        return {"volatility": vol.tolist(), "aic": float(res.aic), "bic": float(res.bic)}
