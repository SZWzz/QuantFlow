"""
Crypto Extra Data Wrapper
DeFi Llama TVL, Etherscan whale/gas data
"""

import sys
import json
import requests
from typing import Dict, Any


class CryptoExtrasWrapper:
    def __init__(self):
        self.timeout = 15

    def get_defi_tvl(self) -> Dict[str, Any]:
        """Top DeFi protocols by TVL from DeFi Llama (free, no API key)."""
        try:
            resp = requests.get("https://api.llama.fi/protocols", timeout=self.timeout)
            if resp.status_code != 200:
                return {"success": False, "error": f"HTTP {resp.status_code}"}
            data = resp.json()
            data.sort(key=lambda p: p.get("tvl", 0), reverse=True)
            return {
                "success": True,
                "data": data[:100],
                "count": min(len(data), 100),
                "timestamp": __import__('time').time(),
                "source": "defillama"
            }
        except Exception as e:
            return {"success": False, "error": str(e), "data": [], "count": 0, "timestamp": __import__('time').time()}

    def get_whale_transactions(self, address: str = "") -> Dict[str, Any]:
        """Large ETH/ERC-20 transfers from Etherscan (free API key needed)."""
        api_key = __import__('os').environ.get("ETHERSCAN_API_KEY", "")
        if not api_key:
            return {"success": False, "error": "ETHERSCAN_API_KEY not set", "data": [], "count": 0}
        try:
            if address:
                url = f"https://api.etherscan.io/api?module=account&action=tokentx&address={address}&sort=desc&offset=50&apikey={api_key}"
            else:
                url = f"https://api.etherscan.io/api?module=account&action=tokentx&address=0x000000000000000000000000000000000000dead&sort=desc&offset=50&apikey={api_key}"
            resp = requests.get(url, timeout=self.timeout)
            if resp.status_code != 200:
                return {"success": False, "error": f"HTTP {resp.status_code}"}
            data = resp.json()
            if data.get("status") != "1":
                return {"success": False, "error": data.get("message", "unknown"), "data": [], "count": 0}
            result = data.get("result", [])
            whaled = [t for t in result if float(t.get("value", 0)) / 10**18 * 2000 > 1000000]
            return {
                "success": True,
                "data": whaled[:50],
                "count": len(whaled),
                "timestamp": __import__('time').time(),
                "source": "etherscan"
            }
        except Exception as e:
            return {"success": False, "error": str(e), "data": [], "count": 0}

    def get_gas_fees(self) -> Dict[str, Any]:
        """Current gas fees from Etherscan Gas Tracker."""
        api_key = __import__('os').environ.get("ETHERSCAN_API_KEY", "")
        if not api_key:
            return {"success": False, "error": "ETHERSCAN_API_KEY not set"}
        try:
            url = f"https://api.etherscan.io/api?module=gastracker&action=gasoracle&apikey={api_key}"
            resp = requests.get(url, timeout=self.timeout)
            if resp.status_code != 200:
                return {"success": False, "error": f"HTTP {resp.status_code}"}
            data = resp.json()
            if data.get("status") != "1":
                return {"success": False, "error": data.get("message", "unknown")}
            return {
                "success": True,
                "data": data.get("result", {}),
                "count": 1,
                "timestamp": __import__('time').time(),
                "source": "etherscan"
            }
        except Exception as e:
            return {"success": False, "error": str(e)}

    def get_depth(self, exchange: str = "binance", symbol: str = "BTC/USDT", limit: int = 50) -> Dict[str, Any]:
        """Order book depth via CCXT."""
        try:
            from src.data.fincept.ccxt_client import make_exchange
            ex = make_exchange(exchange)
            ob = ex.fetch_order_book(symbol, limit=limit)
            return {
                "success": True,
                "data": {
                    "bids": ob.get("bids", [])[:limit],
                    "asks": ob.get("asks", [])[:limit],
                    "symbol": symbol,
                    "exchange": exchange,
                },
                "timestamp": __import__('time').time(),
                "source": f"ccxt.{exchange}"
            }
        except Exception as e:
            return {"success": False, "error": str(e), "data": {}}


def main():
    wrapper = CryptoExtrasWrapper()
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: crypto_extras.py <endpoint> [args...]"}))
        return
    endpoint = sys.argv[1]
    if endpoint == "get_defi_tvl":
        result = wrapper.get_defi_tvl()
    elif endpoint == "get_whale_transactions":
        address = sys.argv[2] if len(sys.argv) > 2 else ""
        result = wrapper.get_whale_transactions(address)
    elif endpoint == "get_gas_fees":
        result = wrapper.get_gas_fees()
    elif endpoint == "get_depth":
        exchange = sys.argv[2] if len(sys.argv) > 2 else "binance"
        symbol = sys.argv[3] if len(sys.argv) > 3 else "BTC/USDT"
        limit = int(sys.argv[4]) if len(sys.argv) > 4 else 50
        result = wrapper.get_depth(exchange, symbol, limit)
    else:
        print(json.dumps({"error": f"Unknown endpoint: {endpoint}"}))
        return
    print(json.dumps(result, ensure_ascii=True, default=str))


if __name__ == "__main__":
    main()
