"""A 股选股扫描 CLI — 基于 akshare 实时数据 + K 线分析的批量扫描。"""

import json
import os
import sys
import traceback
import akshare as ak
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from concurrent.futures import ThreadPoolExecutor, as_completed

# Suppress akshare progress bars
os.environ["AKSHARE_RAISE_ERR"] = "False"


def _fetch_ohlcv(symbol: str) -> pd.DataFrame | None:
    """获取单只股票的日 K 线（返回最近 120 个交易日）。"""
    try:
        end = datetime.now()
        start = end - timedelta(days=365)
        df = ak.stock_zh_a_hist(
            symbol=symbol, period="daily",
            start_date=start.strftime("%Y%m%d"),
            end_date=end.strftime("%Y%m%d"), adjust="qfq",
        )
        if df.empty or len(df) < 30:
            return None
        df.rename(columns={
            "日期": "datetime", "开盘": "open", "收盘": "close",
            "最高": "high", "最低": "low", "成交量": "volume",
        }, inplace=True)
        df["datetime"] = pd.to_datetime(df["datetime"])
        for c in ["open", "close", "high", "low", "volume"]:
            df[c] = pd.to_numeric(df[c], errors="coerce")
        return df
    except Exception:
        return None


class Scanner:
    """批量扫描器 — 多策略选股。"""

    STRATEGIES = {
        "golden_cross": "金叉选股",
        "macd_golden": "MACD金叉",
        "volume_break": "放量突破",
        "breakout_high": "突破前高",
        "oversold_bounce": "超跌反弹",
        "bullish_engulf": "看涨吞没",
        "ma_support": "均线支撑",
        "multi_factor": "多因子综合",
    }

    def scan(self, strategy_name: str, top_n: int = 50) -> dict:
        """执行扫描并返回排名结果。"""
        spot = self._get_spot_data()
        if spot is None or spot.empty:
            return {"strategy": strategy_name, "results": [], "scanned": 0}

        all_symbols = spot["代码"].tolist()
        scanned = len(all_symbols)
        name_map = dict(zip(spot["代码"], spot["名称"]))

        top_symbols = all_symbols[:200]
        ohlcv_cache: dict[str, pd.DataFrame] = {}
        with ThreadPoolExecutor(max_workers=10) as pool:
            fut = {pool.submit(_fetch_ohlcv, s): s for s in top_symbols}
            for f in as_completed(fut, timeout=60):
                s = fut[f]
                try:
                    df = f.result()
                    if df is not None:
                        ohlcv_cache[s] = df
                except Exception:
                    pass

        results = []
        for sym in all_symbols:
            name = name_map.get(sym, "")
            row = spot[spot["代码"] == sym]
            if row.empty:
                continue
            r = row.iloc[0]
            price = float(r.get("最新价", 0))
            change = float(r.get("涨跌幅", 0))
            volume = float(r.get("成交量", 0))

            score, signal, conditions = self._eval_strategy(
                strategy_name, sym, price, change, volume, ohlcv_cache.get(sym),
            )
            if score > 0:
                results.append({
                    "symbol": sym, "name": name,
                    "score": round(score, 2),
                    "signal": signal,
                    "last_price": price,
                    "change_pct": change,
                    "volume": int(volume),
                    "matched_conditions": conditions,
                })

        results.sort(key=lambda x: x["score"], reverse=True)
        return {
            "strategy": strategy_name,
            "results": results[:top_n],
            "scanned": scanned,
        }

    def _get_spot_data(self) -> pd.DataFrame | None:
        """获取 A 股实时行情（主力资金排序取前 2000 只）。"""
        try:
            df = ak.stock_zh_a_spot_em()
            if df.empty:
                return None
            df = df.sort_values("成交额", ascending=False).head(2000)
            return df
        except Exception:
            return None

    def _eval_strategy(
        self, strategy: str, sym: str,
        price: float, change: float, volume: float,
        ohlcv: pd.DataFrame | None,
    ) -> tuple[float, str, list[str]]:
        """评估单只股票是否符合策略条件。返回 (score, signal, conditions)。"""
        conditions: list[str] = []

        if strategy == "golden_cross":
            if ohlcv is None or len(ohlcv) < 30:
                return 0, "", conditions
            ma5 = ohlcv["close"].rolling(5).mean().iloc[-1]
            ma20 = ohlcv["close"].rolling(20).mean().iloc[-1]
            ma5_prev = ohlcv["close"].rolling(5).mean().iloc[-2]
            ma20_prev = ohlcv["close"].rolling(20).mean().iloc[-2]
            if ma5_prev <= ma20_prev and ma5 > ma20:
                conditions.append("5日线上穿20日线")
                score = 50 + min(50, (ma5 / ma20 - 1) * 1000)
                return score, "买入", conditions
            if ma5 > ma20:
                conditions.append("5日线在20日线上方")
                score = 30 + (ma5 / ma20 - 1) * 500
                return score, "持有", conditions
            return 0, "", conditions

        if strategy == "macd_golden":
            if ohlcv is None or len(ohlcv) < 26:
                return 0, "", conditions
            close = ohlcv["close"].values
            ema12 = pd.Series(close).ewm(span=12).mean().values
            ema26 = pd.Series(close).ewm(span=26).mean().values
            dif = ema12 - ema26
            dea = pd.Series(dif).ewm(span=9).mean().values
            if len(dif) >= 2 and dif[-2] <= dea[-2] and dif[-1] > dea[-1]:
                conditions.append("MACD金叉")
                score = 50 + min(50, abs(dif[-1] - dea[-1]) * 100)
                return score, "买入", conditions
            if dif[-1] > dea[-1]:
                conditions.append("DIF在DEA上方")
                score = 25 + (dif[-1] - dea[-1]) * 200
                return score, "持有", conditions
            return 0, "", conditions

        if strategy == "volume_break":
            if ohlcv is None or len(ohlcv) < 20:
                return 0, "", conditions
            avg_vol = ohlcv["volume"].tail(20).mean()
            if avg_vol <= 0:
                return 0, "", conditions
            ratio = volume / avg_vol
            if ratio >= 1.5 and change > 0:
                conditions.append(f"成交量突破 {ratio:.1f} 倍于20日均量")
                conditions.append("价格上涨")
                score = 40 + min(60, ratio * 20)
                return score, "买入", conditions
            if ratio >= 1.2:
                conditions.append(f"成交量 {ratio:.1f} 倍于20日均量")
                score = 15 + ratio * 10
                return score, "关注", conditions
            return 0, "", conditions

        if strategy == "breakout_high":
            if ohlcv is None or len(ohlcv) < 60:
                return 0, "", conditions
            high_60 = ohlcv["high"].tail(60).max()
            recent_high = ohlcv["high"].tail(5).max()
            if recent_high >= high_60 and change > 0:
                conditions.append("突破60日最高价")
                score = 50 + min(50, (price / high_60 - 1) * 500)
                return score, "买入", conditions
            if price >= high_60 * 0.98:
                conditions.append("接近60日最高价")
                score = 20 + (price / high_60 - 0.98) * 500
                return score, "关注", conditions
            return 0, "", conditions

        if strategy == "oversold_bounce":
            if ohlcv is None or len(ohlcv) < 14:
                return 0, "", conditions
            close = ohlcv["close"].values
            high = ohlcv["high"].values
            low = ohlcv["low"].values
            rsi_period = 14
            diff = np.diff(close)
            gain = np.where(diff > 0, diff, 0).mean()
            loss = np.where(diff < 0, -diff, 0).mean()
            if loss == 0:
                return 0, "", conditions
            rs = gain / loss
            rsi = 100 - 100 / (1 + rs)
            if rsi < 30 and change > 0:
                conditions.append(f"RSI={rsi:.1f} 超卖区反弹")
                score = 50 + min(50, (30 - rsi) * 2)
                return score, "买入", conditions
            if rsi < 40:
                conditions.append(f"RSI={rsi:.1f} 接近超卖区")
                score = 15 + (40 - rsi)
                return score, "关注", conditions
            return 0, "", conditions

        if strategy == "bullish_engulf":
            if ohlcv is None or len(ohlcv) < 3:
                return 0, "", conditions
            last2 = ohlcv.tail(2)
            if len(last2) < 2:
                return 0, "", conditions
            prev = last2.iloc[0]
            curr = last2.iloc[1]
            if prev["close"] < prev["open"] and curr["close"] > curr["open"]:
                if curr["open"] <= prev["close"] and curr["close"] >= prev["open"]:
                    conditions.append("阳线实体吞没前日阴线")
                    score = 60 + (curr["close"] - curr["open"]) / curr["open"] * 500
                    return score, "买入", conditions
            return 0, "", conditions

        if strategy == "ma_support":
            if ohlcv is None or len(ohlcv) < 60:
                return 0, "", conditions
            ma = ohlcv["close"].rolling(60).mean()
            if ma.isna().all():
                return 0, "", conditions
            ma_val = ma.iloc[-1]
            if ma_val <= 0:
                return 0, "", conditions
            deviation = (price - ma_val) / ma_val
            if -0.02 <= deviation <= 0.01 and change > 0:
                conditions.append(f"回踩60日线不破 ({deviation:+.2%})")
                score = 50 + max(0, (0.02 - abs(deviation)) * 1000)
                return score, "买入", conditions
            if -0.04 <= deviation < -0.02:
                conditions.append(f"接近60日线 ({deviation:+.2%})")
                score = 20 + (0.04 - abs(deviation)) * 500
                return score, "观察", conditions
            return 0, "", conditions

        if strategy == "multi_factor":
            factors = []
            score = 0
            if change > 3:
                factors.append(f"涨幅 {change:.1f}%")
                score += 20
            elif change > 1:
                factors.append(f"涨幅 {change:.1f}%")
                score += 10
            if volume > 1e8:
                factors.append("成交额 > 1亿")
                score += 15
            if ohlcv is not None and len(ohlcv) >= 20:
                ma5 = ohlcv["close"].rolling(5).mean().iloc[-1]
                ma20 = ohlcv["close"].rolling(20).mean().iloc[-1]
                if ma5 > ma20:
                    factors.append("5日线 > 20日线")
                    score += 15
                avg_vol = ohlcv["volume"].tail(20).mean()
                if avg_vol > 0 and volume / avg_vol > 1.5:
                    factors.append("放量")
                    score += 15
                high_20 = ohlcv["high"].tail(20).max()
                if high_20 > 0 and price / high_20 > 0.95:
                    factors.append("接近20日高点")
                    score += 10
            if factors:
                return min(score, 100), "综合推荐", factors
            return 0, "", conditions

        return 0, "", conditions


def main():
    if len(sys.argv) < 2:
        print(json.dumps({
            "error": "Usage: python -m src.data.fincept.scanner <strategy> [top_n]",
            "strategies": Scanner.STRATEGIES,
        }))
        return
    strategy = sys.argv[1].strip()
    top_n = int(sys.argv[2]) if len(sys.argv) > 2 else 50

    if strategy not in Scanner.STRATEGIES:
        print(json.dumps({"error": f"未知策略: {strategy}，可用: {list(Scanner.STRATEGIES.keys())}", "results": [], "scanned": 0}))
        return

    try:
        scanner = Scanner()
        result = scanner.scan(strategy, top_n)
        print(json.dumps(result, ensure_ascii=False, default=str))
    except Exception as e:
        print(json.dumps({"error": str(e), "traceback": traceback.format_exc(), "results": [], "scanned": 0}, ensure_ascii=False))


if __name__ == "__main__":
    main()
