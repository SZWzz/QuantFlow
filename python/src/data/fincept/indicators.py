"""技术指标计算 CLI — 通过 akshare 获取 K 线 → 指标计算 → JSON 输出。"""

import json
import os
import sys
import traceback
import akshare as ak
import pandas as pd
from datetime import datetime, timedelta

os.environ["AKSHARE_RAISE_ERR"] = "False"


def compute_indicator(symbol: str, indicator_name: str, params_json: str = "{}") -> dict:
    """对 A 股标的计算指定技术指标。"""
    from src.indicators import compute_indicators, list_indicators

    params = json.loads(params_json) if params_json else {}

    end = datetime.now()
    start = end - timedelta(days=365 * 2)
    df = ak.stock_zh_a_hist(
        symbol=symbol,
        period="daily",
        start_date=start.strftime("%Y%m%d"),
        end_date=end.strftime("%Y%m%d"),
        adjust="qfq",
    )
    if df.empty:
        return {"error": f"未获取到 {symbol} 的 K 线数据", "symbol": symbol, "data": []}

    df.rename(columns={
        "日期": "datetime", "开盘": "open", "收盘": "close",
        "最高": "high", "最低": "low", "成交量": "volume",
    }, inplace=True)
    df["datetime"] = pd.to_datetime(df["datetime"]).astype(str)

    name_upper = indicator_name.upper()
    all_indicators = {spec["name"]: spec for spec in list_indicators()}
    if name_upper not in all_indicators:
        return {
            "error": f"未知指标: {indicator_name}。可用: {sorted(all_indicators.keys())}",
            "symbol": symbol, "data": [],
        }

    try:
        result_df = compute_indicators(df, [name_upper], params={name_upper: params})
    except Exception as e:
        return {"error": str(e), "symbol": symbol, "data": []}

    if result_df.empty:
        return {"symbol": symbol, "indicator": indicator_name, "data": []}

    out = result_df.dropna(how="all").tail(120).copy()
    out = out.where(pd.notna(out), None)
    records = out.to_dict(orient="records")
    return {
        "symbol": symbol,
        "indicator": indicator_name,
        "params": params,
        "data": records,
    }


def main():
    if len(sys.argv) < 3:
        print(json.dumps({
            "error": "Usage: python -m src.data.fincept.indicators <symbol> <indicator_name> [params_json]",
            "data": [],
        }))
        return
    symbol = sys.argv[1].strip()
    indicator_name = sys.argv[2].strip()
    params_json = sys.argv[3] if len(sys.argv) > 3 else "{}"
    try:
        result = compute_indicator(symbol, indicator_name, params_json)
        print(json.dumps(result, ensure_ascii=False, default=str))
    except Exception as e:
        print(json.dumps({"error": str(e), "traceback": traceback.format_exc(), "data": []}, ensure_ascii=False))


if __name__ == "__main__":
    main()
