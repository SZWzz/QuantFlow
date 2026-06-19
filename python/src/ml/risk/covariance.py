"""Covariance matrix estimation."""
import numpy as np
import pyarrow as pa

_HAS_SKLEARN = False
try:
    from sklearn.covariance import LedoitWolf
    _HAS_SKLEARN = True
except ImportError:
    pass


class CovarianceEngine:
    """Covariance matrix estimation for multi-asset returns.

    Supports Ledoit-Wolf shrinkage and sample covariance.
    """

    def estimate(self, returns: pa.Table, params: dict) -> dict:
        """Estimate a covariance matrix from multi-asset returns.

        Args:
            returns: Arrow Table with one column per asset.
            params: dict with keys: method ("ledoit_wolf" or "sample").

        Returns:
            dict with keys: covariance (list of lists), method (str).
        """
        method = params.get("method", "ledoit_wolf")
        df = returns.to_pandas()
        R = df.values.astype(np.float64)

        if method == "ledoit_wolf":
            if not _HAS_SKLEARN:
                raise ImportError("scikit-learn is required for LedoitWolf. Install with: pip install scikit-learn")
            lw = LedoitWolf().fit(R)
            cov = lw.covariance_
        else:
            cov = np.cov(R, rowvar=False)

        return {"covariance": cov.tolist(), "method": method}
