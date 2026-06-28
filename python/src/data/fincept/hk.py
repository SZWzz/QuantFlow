"""
AKShare Hong Kong Market Data Wrapper
Wrapper for HK stock market data: IPO subscription/listing, CBBC, warrants, trading calendar
Returns JSON output for Go integration
"""

import sys
import json
import pandas as pd
import akshare as ak
from typing import Dict, Any, List
from datetime import datetime, date


class DateTimeEncoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, (datetime, date)):
            return obj.isoformat()
        return super().default(obj)


class AKShareError:
    def __init__(self, endpoint: str, error: str, data_source: str = None):
        self.endpoint = endpoint
        self.error = error
        self.data_source = data_source
        self.timestamp = int(datetime.now().timestamp())

    def to_dict(self) -> Dict[str, Any]:
        return {
            "endpoint": self.endpoint,
            "error": self.error,
            "data_source": self.data_source,
            "timestamp": self.timestamp,
            "type": "AKShareError"
        }


class HKWrapper:
    def __init__(self):
        self.default_timeout = 30
        self.retry_delay = 2

    def _convert_dataframe_to_json_safe(self, df: pd.DataFrame) -> List[Dict[str, Any]]:
        df_copy = df.copy()
        for col in df_copy.columns:
            if pd.api.types.is_datetime64_any_dtype(df_copy[col]):
                df_copy[col] = df_copy[col].astype(str)
        return df_copy.to_dict('records')

    def _safe_call_with_retry(self, func, *args, max_retries: int = 3, **kwargs) -> Dict[str, Any]:
        last_error = None
        for attempt in range(max_retries):
            try:
                result = func(*args, **kwargs)
                if result is not None and hasattr(result, 'empty') and not result.empty:
                    return {
                        "success": True,
                        "data": self._convert_dataframe_to_json_safe(result),
                        "count": len(result),
                        "timestamp": int(datetime.now().timestamp()),
                        "source": f"akshare.{func.__name__}"
                    }
                return {
                    "success": False,
                    "data": [],
                    "count": 0,
                    "error": f"Empty result from {func.__name__}",
                    "timestamp": int(datetime.now().timestamp())
                }
            except Exception as e:
                last_error = str(e)
                import time
                time.sleep(self.retry_delay * (attempt + 1))
        return {
            "success": False,
            "data": [],
            "count": 0,
            "error": f"Failed after {max_retries} retries: {last_error}",
            "timestamp": int(datetime.now().timestamp())
        }

    def _safe_call_list(self, func, *args, max_retries: int = 3, **kwargs) -> Dict[str, Any]:
        last_error = None
        for attempt in range(max_retries):
            try:
                result = func(*args, **kwargs)
                if result is not None and isinstance(result, list) and len(result) > 0:
                    return {
                        "success": True,
                        "data": result,
                        "count": len(result),
                        "timestamp": int(datetime.now().timestamp()),
                        "source": f"akshare.{func.__name__}"
                    }
                return {
                    "success": False,
                    "data": [],
                    "count": 0,
                    "error": f"Empty list result from {func.__name__}",
                    "timestamp": int(datetime.now().timestamp())
                }
            except Exception as e:
                last_error = str(e)
                import time
                time.sleep(self.retry_delay * (attempt + 1))
        return {
            "success": False,
            "data": [],
            "count": 0,
            "error": f"Failed after {max_retries} retries: {last_error}",
            "timestamp": int(datetime.now().timestamp())
        }

    def get_hk_ipo_subscription(self) -> Dict[str, Any]:
        """Get HK IPO subscription data (新股认购)"""
        return self._safe_call_with_retry(ak.stock_hk_ipo_subscription)

    def get_hk_ipo_record(self) -> Dict[str, Any]:
        """Get HK IPO listing records (新股上市表现)"""
        return self._safe_call_with_retry(ak.stock_hk_ipo_record)

    def get_hk_cbbc(self) -> Dict[str, Any]:
        """Get HK CBBC data (牛熊证)"""
        return self._safe_call_with_retry(ak.stock_hk_cbbc)

    def get_hk_warrants(self) -> Dict[str, Any]:
        """Get HK warrants data (认股证/涡轮)"""
        return self._safe_call_with_retry(ak.stock_hk_warrants)

    def get_hk_trade_cal(self) -> Dict[str, Any]:
        """Get HK trading calendar (交易日历)"""
        return self._safe_call_with_retry(ak.tool_trade_date_hist)


def main():
    wrapper = HKWrapper()

    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: python hk.py <endpoint>"}))
        return

    endpoint = sys.argv[1]
    method_name = f"get_{endpoint}" if not endpoint.startswith("get_") else endpoint

    if hasattr(wrapper, method_name):
        method = getattr(wrapper, method_name)
        result = method()
        print(json.dumps(result, ensure_ascii=True, cls=DateTimeEncoder))
    else:
        print(json.dumps({"error": f"Unknown endpoint: {endpoint}"}))


if __name__ == "__main__":
    main()
