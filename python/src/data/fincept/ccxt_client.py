"""
Unified exchange client — shared initialization for all exchange scripts.
Handles exchange instantiation, credential loading, and error normalization.

Called by C++ python_runner as subprocess — outputs JSON to stdout.
"""

import os
import sys
import json
import time
import ccxt

# Markets cache TTL — 30 minutes is a safe balance between freshness and speed.
# Exchange market lists rarely change within a session.
_MARKETS_CACHE_TTL = 1800


def get_markets_cache_path(exchange_id: str) -> str:
    """Return path to the per-exchange markets cache file."""
    cache_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".cache")
    os.makedirs(cache_dir, exist_ok=True)
    return os.path.join(cache_dir, f"markets_{exchange_id}.json")


def load_cached_markets(exchange_id: str) -> dict | None:
    """Return cached markets dict if still fresh (within TTL), else None."""
    path = get_markets_cache_path(exchange_id)
    try:
        if os.path.exists(path):
            age = time.time() - os.path.getmtime(path)
            if age < _MARKETS_CACHE_TTL:
                with open(path, "r", encoding="utf-8") as f:
                    return json.load(f)
    except Exception:
        pass
    return None


def save_markets_cache(exchange_id: str, markets: dict) -> None:
    """Persist markets dict to disk so the next WS startup can skip load_markets()."""
    path = get_markets_cache_path(exchange_id)
    try:
        with open(path, "w", encoding="utf-8") as f:
            json.dump(markets, f)
    except Exception:
        pass


def get_default_type(exchange_id: str) -> str:
    """Return the appropriate defaultType for an exchange.
    Hyperliquid is a perps DEX — its primary market type is swap.
    All others default to spot.
    """
    return "swap" if exchange_id in ("hyperliquid",) else "spot"


def make_exchange(exchange_id: str, credentials: dict = None,
                  sandbox: bool = False, timeout_ms: int = 10000) -> ccxt.Exchange:
    """Create and configure a ccxt exchange instance.

    timeout_ms defaults to 10s (was 30s) — fast failure on slow exchanges.
    WS stream sets its own timeout directly on the ccxt.pro instance.
    """
    if exchange_id not in ccxt.exchanges:
        raise ValueError(f"Unknown exchange: {exchange_id}. Available: {len(ccxt.exchanges)}")

    config = {
        "enableRateLimit": True,
        "timeout": timeout_ms,
        "options": {"defaultType": get_default_type(exchange_id)},
    }

    if credentials:
        if credentials.get("api_key"):
            config["apiKey"] = credentials["api_key"]
        if credentials.get("secret"):
            config["secret"] = credentials["secret"]
        if credentials.get("password"):
            config["password"] = credentials["password"]
        if credentials.get("uid"):
            config["uid"] = credentials["uid"]

    exchange_class = getattr(ccxt, exchange_id)
    exchange = exchange_class(config)

    if sandbox:
        exchange.set_sandbox_mode(True)

    return exchange


def output_success(data: dict | list):
    """Print JSON result to stdout for C++ python_runner to extract.
    Uses compact separators to reduce payload size (no whitespace)."""
    print(json.dumps({"success": True, "data": data}, default=str, separators=(",", ":")))


def output_error(message: str, code: str = "EXCHANGE_ERROR"):
    """Print JSON error to stdout."""
    print(json.dumps({"success": False, "error": message, "code": code}, default=str, separators=(",", ":")))
    sys.exit(1)


def parse_credentials_from_stdin() -> dict:
    """Read credentials from stdin (passed by C++ via execute_with_stdin)."""
    try:
        data = sys.stdin.read()
        if data.strip():
            return json.loads(data)
    except (json.JSONDecodeError, EOFError):
        pass
    return {}


def run_with_error_handling(func):
    """Decorator to catch ccxt exceptions and output normalized errors."""
    def wrapper(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except ccxt.AuthenticationError as e:
            output_error(f"Authentication failed: {e}", "AUTH_ERROR")
        except ccxt.InsufficientFunds as e:
            output_error(f"Insufficient funds: {e}", "INSUFFICIENT_FUNDS")
        except ccxt.InvalidOrder as e:
            output_error(f"Invalid order: {e}", "INVALID_ORDER")
        except ccxt.OrderNotFound as e:
            output_error(f"Order not found: {e}", "ORDER_NOT_FOUND")
        except ccxt.RateLimitExceeded as e:
            output_error(f"Rate limit exceeded: {e}", "RATE_LIMIT")
        except ccxt.NetworkError as e:
            output_error(f"Network error: {e}", "NETWORK_ERROR")
        except ccxt.ExchangeNotAvailable as e:
            output_error(f"Exchange unavailable: {e}", "EXCHANGE_UNAVAILABLE")
        except ccxt.ExchangeError as e:
            output_error(f"Exchange error: {e}", "EXCHANGE_ERROR")
        except Exception as e:
            output_error(f"Unexpected error: {e}", "UNKNOWN_ERROR")
    return wrapper
"""
Fetch OHLCV candlestick data for charting and technical analysis.

Usage: python fetch_ohlcv.py <exchange_id> <symbol> [timeframe] [limit]
Example: python fetch_ohlcv.py binance BTC/USDT 1h 100

Output JSON:
{
  "success": true,
  "data": {
    "symbol": "BTC/USDT",
    "timeframe": "1h",
    "candles": [
      {"timestamp": 1773457200000, "open": 70783.27, "high": 71185.23, "low": 70783.26, "close": 71081.3, "volume": 944.72},
      ...
    ],
    "count": 100
  }
}
"""

import sys
# exchange_client functions are defined above in this merged file


@run_with_error_handling
def main():
    if len(sys.argv) < 3:
        output_error("Usage: fetch_ohlcv.py <exchange_id> <symbol> [timeframe] [limit]", "INVALID_ARGS")

    exchange_id = sys.argv[1]
    symbol = sys.argv[2]
    timeframe = sys.argv[3] if len(sys.argv) > 3 else "1h"
    limit = int(sys.argv[4]) if len(sys.argv) > 4 else 100

    exchange = make_exchange(exchange_id)
    candles = exchange.fetch_ohlcv(symbol, timeframe, limit=limit)

    output_success({
        "symbol": symbol,
        "timeframe": timeframe,
        "candles": [
            {
                "timestamp": c[0],
                "open": c[1],
                "high": c[2],
                "low": c[3],
                "close": c[4],
                "volume": c[5],
            }
            for c in candles
        ],
        "count": len(candles),
    })


if __name__ == "__main__":
    main()
"""
Fetch ticker (last price, bid, ask, volume) for a symbol.
Used by C++ OrderMatcher to get live prices for paper trading fills.

Usage: python fetch_ticker.py <exchange_id> <symbol>
Example: python fetch_ticker.py binance BTC/USDT
         python fetch_ticker.py kraken ETH/USD

Output JSON:
{
  "success": true,
  "data": {
    "symbol": "BTC/USDT",
    "last": 70990.5,
    "bid": 70990.49,
    "ask": 70990.5,
    "high": 73913.74,
    "low": 70481.7,
    "open": 71568.0,
    "close": 70990.5,
    "change": -577.5,
    "percentage": -0.807,
    "base_volume": 33621.71,
    "quote_volume": 2419989519.12,
    "timestamp": 1773466621013
  }
}
"""

import sys
# exchange_client functions are defined above in this merged file


@run_with_error_handling
def main():
    if len(sys.argv) < 3:
        output_error("Usage: fetch_ticker.py <exchange_id> <symbol>", "INVALID_ARGS")

    exchange_id = sys.argv[1]
    symbol = sys.argv[2]

    exchange = make_exchange(exchange_id)
    ticker = exchange.fetch_ticker(symbol)

    output_success({
        "symbol": ticker["symbol"],
        "last": ticker.get("last"),
        "bid": ticker.get("bid"),
        "ask": ticker.get("ask"),
        "bid_volume": ticker.get("bidVolume"),
        "ask_volume": ticker.get("askVolume"),
        "high": ticker.get("high"),
        "low": ticker.get("low"),
        "open": ticker.get("open"),
        "close": ticker.get("close"),
        "change": ticker.get("change"),
        "percentage": ticker.get("percentage"),
        "vwap": ticker.get("vwap"),
        "base_volume": ticker.get("baseVolume"),
        "quote_volume": ticker.get("quoteVolume"),
        "timestamp": ticker.get("timestamp"),
    })


if __name__ == "__main__":
    main()
"""
Fetch order book (bids/asks) for a symbol.
Used for depth visualization and slippage estimation.

Usage: python fetch_orderbook.py <exchange_id> <symbol> [limit]
Example: python fetch_orderbook.py binance BTC/USDT 20

Output JSON:
{
  "success": true,
  "data": {
    "symbol": "BTC/USDT",
    "bids": [[70990.49, 1.33], [70990.48, 0.0004], ...],
    "asks": [[70990.50, 1.56], [70990.51, 0.00079], ...],
    "timestamp": 1773466621013,
    "best_bid": 70990.49,
    "best_ask": 70990.50,
    "spread": 0.01,
    "spread_pct": 0.000014
  }
}
"""

import sys
# exchange_client functions are defined above in this merged file


@run_with_error_handling
def main():
    if len(sys.argv) < 3:
        output_error("Usage: fetch_orderbook.py <exchange_id> <symbol> [limit]", "INVALID_ARGS")

    exchange_id = sys.argv[1]
    symbol = sys.argv[2]
    limit = int(sys.argv[3]) if len(sys.argv) > 3 else 20

    exchange = make_exchange(exchange_id)
    ob = exchange.fetch_order_book(symbol, limit=limit)

    best_bid = ob["bids"][0][0] if ob["bids"] else 0
    best_ask = ob["asks"][0][0] if ob["asks"] else 0
    spread = best_ask - best_bid if best_bid and best_ask else 0
    spread_pct = (spread / best_ask * 100) if best_ask else 0

    output_success({
        "symbol": ob.get("symbol", symbol),
        "bids": ob["bids"],
        "asks": ob["asks"],
        "timestamp": ob.get("timestamp"),
        "best_bid": best_bid,
        "best_ask": best_ask,
        "spread": spread,
        "spread_pct": round(spread_pct, 6),
    })


if __name__ == "__main__":
    main()
"""
Fetch current funding rate for a perpetual futures symbol.
Works with both spot symbols (BTC/USDT) and perp symbols (BTC/USDT:USDT).
For spot symbols on exchanges that use unified format, tries the perp equivalent.

Usage: python fetch_funding_rate.py <exchange_id> <symbol>
"""
import sys
# exchange_client functions are defined above in this merged file


def to_perp_symbol(symbol: str) -> str:
    """Convert spot symbol to linear perp equivalent if needed.
    BTC/USDT -> BTC/USDT:USDT (Binance unified format)
    """
    if ":" not in symbol and "/" in symbol:
        quote = symbol.split("/")[1]
        return f"{symbol}:{quote}"
    return symbol


@run_with_error_handling
def main():
    if len(sys.argv) < 3:
        output_success({
            "symbol": "",
            "funding_rate": None,
            "funding_timestamp": None,
            "next_funding_timestamp": None,
            "mark_price": None,
            "index_price": None,
        })
        return

    exchange_id = sys.argv[1]
    symbol = sys.argv[2]

    exchange = make_exchange(exchange_id)
    exchange.load_markets()

    # Try perp symbol first, fall back to original
    perp_symbol = to_perp_symbol(symbol)
    tried = []

    for sym in ([perp_symbol, symbol] if perp_symbol != symbol else [symbol]):
        tried.append(sym)
        try:
            # Use swap market type for perp symbols
            exchange.options["defaultType"] = "swap"
            rate = exchange.fetch_funding_rate(sym)
            output_success({
                "symbol": sym,
                "funding_rate": rate.get("fundingRate"),
                "funding_timestamp": rate.get("fundingTimestamp"),
                "next_funding_timestamp": rate.get("nextFundingTimestamp"),
                "mark_price": rate.get("markPrice"),
                "index_price": rate.get("indexPrice"),
                "interest_rate": rate.get("interestRate"),
            })
            return
        except Exception:
            continue

    # Not available for this symbol/exchange (common for spot-only exchanges)
    output_success({
        "symbol": symbol,
        "funding_rate": None,
        "funding_timestamp": None,
        "next_funding_timestamp": None,
        "mark_price": None,
        "index_price": None,
    })


if __name__ == "__main__":
    main()
"""
Fetch open interest for a perpetual futures symbol.
Works with both spot symbols (BTC/USDT) and perp symbols (BTC/USDT:USDT).

Usage: python fetch_open_interest.py <exchange_id> <symbol>
"""
import sys
# exchange_client functions are defined above in this merged file


def to_perp_symbol(symbol: str) -> str:
    """Convert spot symbol to linear perp equivalent if needed."""
    if ":" not in symbol and "/" in symbol:
        quote = symbol.split("/")[1]
        return f"{symbol}:{quote}"
    return symbol


@run_with_error_handling
def main():
    if len(sys.argv) < 3:
        output_success({"symbol": "", "open_interest": None, "open_interest_value": None, "timestamp": None})
        return

    exchange_id = sys.argv[1]
    symbol = sys.argv[2]

    exchange = make_exchange(exchange_id)
    exchange.load_markets()

    perp_symbol = to_perp_symbol(symbol)

    for sym in ([perp_symbol, symbol] if perp_symbol != symbol else [symbol]):
        try:
            exchange.options["defaultType"] = "swap"
            oi = exchange.fetch_open_interest(sym)
            output_success({
                "symbol": sym,
                "open_interest": oi.get("openInterestAmount"),
                "open_interest_value": oi.get("openInterestValue"),
                "timestamp": oi.get("timestamp"),
            })
            return
        except Exception:
            continue

    # Not available for this symbol/exchange
    output_success({
        "symbol": symbol,
        "open_interest": None,
        "open_interest_value": None,
        "timestamp": None,
    })


if __name__ == "__main__":
    main()
"""
Fetch all available trading pairs/markets from an exchange.
Used to populate symbol search and validate trading pairs.

Usage: python fetch_markets.py <exchange_id> [type]
Example: python fetch_markets.py binance spot
         python fetch_markets.py binance swap

Output JSON:
{
  "success": true,
  "data": {
    "exchange": "binance",
    "markets": [
      {
        "symbol": "BTC/USDT",
        "base": "BTC",
        "quote": "USDT",
        "type": "spot",
        "active": true,
        "maker_fee": 0.001,
        "taker_fee": 0.001,
        "precision_amount": 5,
        "precision_price": 2,
        "min_amount": 0.00001,
        "min_cost": 10.0
      },
      ...
    ],
    "count": 2000
  }
}
"""

import sys
# exchange_client functions are defined above in this merged file


@run_with_error_handling
def main():
    if len(sys.argv) < 2:
        output_error("Usage: fetch_markets.py <exchange_id> [type]", "INVALID_ARGS")

    exchange_id = sys.argv[1]
    market_type = sys.argv[2] if len(sys.argv) > 2 else None

    exchange = make_exchange(exchange_id)
    exchange.load_markets()
    # Persist to disk cache so place_order.py / cancel_order.py can skip load_markets()
    save_markets_cache(exchange_id, exchange.markets)

    markets = []
    for symbol, market in exchange.markets.items():
        if market_type and market.get("type") != market_type:
            continue
        if not market.get("active", True):
            continue

        limits = market.get("limits", {})
        amount_limits = limits.get("amount", {})
        cost_limits = limits.get("cost", {})
        precision = market.get("precision", {})

        markets.append({
            "symbol": market["symbol"],
            "base": market.get("base"),
            "quote": market.get("quote"),
            "type": market.get("type"),
            "active": market.get("active"),
            "maker_fee": market.get("maker"),
            "taker_fee": market.get("taker"),
            "precision_amount": precision.get("amount"),
            "precision_price": precision.get("price"),
            "min_amount": amount_limits.get("min"),
            "min_cost": cost_limits.get("min"),
        })

    output_success({
        "exchange": exchange_id,
        "markets": markets,
        "count": len(markets),
    })


if __name__ == "__main__":
    main()
"""
List all supported exchanges with feature availability.

Usage: python list_exchanges.py

Output JSON:
{
  "success": true,
  "data": {
    "exchanges": [
      {
        "id": "binance",
        "name": "Binance",
        "countries": ["JP"],
        "has_fetch_ticker": true,
        "has_fetch_order_book": true,
        "has_fetch_ohlcv": true,
        "has_create_order": true,
        "has_fetch_balance": true,
        "has_fetch_positions": true,
        "has_set_leverage": true
      },
      ...
    ],
    "count": 110,
    "version": "4.5.28"
  }
}
"""

import ccxt
# exchange_client functions are defined above in this merged file


@run_with_error_handling
def main():
    exchanges = []

    for exchange_id in sorted(ccxt.exchanges):
        try:
            exchange_class = getattr(ccxt, exchange_id)
            ex = exchange_class()
            desc = ex.describe()
            has = ex.has

            exchanges.append({
                "id": exchange_id,
                "name": desc.get("name", exchange_id),
                "countries": desc.get("countries", []),
                "has_fetch_ticker": bool(has.get("fetchTicker")),
                "has_fetch_order_book": bool(has.get("fetchOrderBook")),
                "has_fetch_ohlcv": bool(has.get("fetchOHLCV")),
                "has_create_order": bool(has.get("createOrder")),
                "has_fetch_balance": bool(has.get("fetchBalance")),
                "has_fetch_positions": bool(has.get("fetchPositions")),
                "has_set_leverage": bool(has.get("setLeverage")),
            })
        except Exception:
            exchanges.append({"id": exchange_id, "name": exchange_id, "error": True})

    output_success({
        "exchanges": exchanges,
        "count": len(exchanges),
        "version": ccxt.__version__,
    })


if __name__ == "__main__":
    main()
