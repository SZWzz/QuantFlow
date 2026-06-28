"""缠论分析 CLI — 通过 akshare 获取 K 线 → 缠论分析 → JSON 输出。"""

import json
import os
import sys
import traceback
import akshare as ak
import pandas as pd
from datetime import datetime, timedelta

os.environ["AKSHARE_RAISE_ERR"] = "False"


def analyze_chanlun(symbol: str) -> dict:
    """对 A 股标的执行缠论分析。"""
    from src.chanlun import analyze

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
        return {"available": False, "error": f"未获取到 {symbol} 的 K 线数据", "symbol": symbol}

    df.rename(columns={
        "日期": "datetime", "开盘": "open", "收盘": "close",
        "最高": "high", "最低": "low", "成交量": "vol",
    }, inplace=True)
    df["datetime"] = pd.to_datetime(df["datetime"])

    result = analyze(df)
    serializable = _make_serializable(result)
    serializable["symbol"] = symbol
    serializable["available"] = True
    return serializable


def _make_serializable(d: dict) -> dict:
    """递归将不可 JSON 序列化的类型转为基本类型。"""
    out = {}
    for k, v in d.items():
        if hasattr(v, "__dict__"):
            out[k] = _obj_to_dict(v)
        elif isinstance(v, list):
            out[k] = [_obj_to_dict(i) if hasattr(i, "__dict__") else i for i in v]
        elif isinstance(v, dict):
            out[k] = _make_serializable(v)
        else:
            out[k] = v
    return out


def _obj_to_dict(obj) -> dict:
    d = {}
    for attr in dir(obj):
        if attr.startswith("_"):
            continue
        v = getattr(obj, attr)
        if callable(v):
            continue
        if isinstance(v, (str, int, float, bool, type(None))):
            d[attr] = v
        elif isinstance(v, (datetime,)):
            d[attr] = v.isoformat()
        elif hasattr(v, "__dict__"):
            d[attr] = _obj_to_dict(v)
        else:
            try:
                json.dumps(v)
                d[attr] = v
            except (TypeError, OverflowError):
                d[attr] = str(v)
    return d


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: python -m src.data.fincept.chanlun <symbol>", "available": False}))
        return
    symbol = sys.argv[1].strip()
    try:
        result = analyze_chanlun(symbol)
        print(json.dumps(result, ensure_ascii=False, default=str))
    except Exception as e:
        print(json.dumps({
            "error": str(e), "traceback": traceback.format_exc(),
            "symbol": symbol, "available": False,
        }, ensure_ascii=False))


if __name__ == "__main__":
    main()
