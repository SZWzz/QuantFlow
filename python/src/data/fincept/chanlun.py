"""缠论分析 CLI — 通过 akshare 获取 K 线 → 缠论分析 → JSON 输出。"""

import json
import os
import sys
import traceback
from datetime import datetime, timedelta

import akshare as ak
import pandas as pd

os.environ["AKSHARE_RAISE_ERR"] = "False"


def _fx_type_cn(fx_type_val: str) -> str:
    return "顶分型" if fx_type_val == "ding" else "底分型"


def _direction_symbol(d: str) -> str:
    if d == "up":
        return "↑"
    if d == "down":
        return "↓"
    return "—"


def _fmt_date(dt) -> str:
    if hasattr(dt, "strftime"):
        return dt.strftime("%Y-%m-%d")
    return str(dt)[:10]


def _fractals_to_frontend(fxs: list) -> list[dict]:
    out = []
    for fx in fxs:
        klines = fx.klines
        candle3 = "".join(_direction_symbol(k.direction) for k in klines)
        merged = any(k.merged_count > 1 for k in klines)
        out.append({
            "type": _fx_type_cn(fx.fx_type.value),
            "date": _fmt_date(fx.k.date),
            "candle3": candle3,
            "confirmed": fx.done,
            "merged": merged,
        })
    return out


def _bis_to_frontend(bis: list) -> list[dict]:
    out = []
    for bi in bis:
        from_price = bi.start.val
        to_price = bi.end.val
        pct = (to_price - from_price) / from_price if from_price != 0 else 0.0
        bars = bi.end.k.index - bi.start.k.index + 1
        direction = bi.direction.value
        out.append({
            "from_date": _fmt_date(bi.start.k.date),
            "to_date": _fmt_date(bi.end.k.date),
            "direction": direction,
            "from_price": round(from_price, 2),
            "to_price": round(to_price, 2),
            "pct": round(pct, 4),
            "bars": bars,
        })
    return out


def _zss_to_frontend(zss: list) -> list[dict]:
    out = []
    for zs in zss:
        lines = zs.lines
        direction = lines[0].direction.value if lines else ""

        out.append({
            "from_date": _fmt_date(zs.start.k.date) if zs.start and zs.start.k else "",
            "to_date": _fmt_date(zs.end.k.date) if zs.end and zs.end.k else "",
            "high": round(zs.gg, 2),
            "low": round(zs.dd, 2),
            "direction": direction,
            "bars": zs.line_count,
            "zg": round(zs.zg, 2),
            "zd": round(zs.zd, 2),
        })
    return out


def analyze_chanlun(symbol: str) -> dict:
    """对 A 股标的执行缠论分析，返回前端友好的 JSON 结构。"""
    from src.chanlun import ChanlunAnalyser

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

    analyser = ChanlunAnalyser()
    result = analyser.process_klines(df)

    output: dict = {
        "symbol": symbol,
        "available": True,
        "fractals": _fractals_to_frontend(result.fractals),
        "bi_list": _bis_to_frontend(result.bis),
        "zs_list": _zss_to_frontend(result.zss),
    }

    # Include additional analysis data for future frontend use
    d = result.to_dict()
    output["xd_list"] = d.get("xds", [])
    output["mmd_list"] = d.get("mmds", [])
    output["beichi_list"] = d.get("bcs", [])

    return output


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: python -m src.data.fincept.chanlun <symbol>", "available": False}))
        return
    symbol = sys.argv[1].strip().upper()
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
