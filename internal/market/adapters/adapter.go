// Package adapters provides market data source adapter implementations.
// All adapters implement the market.Adapter interface defined in the parent package.
//
// To add a new adapter:
//  1. Create a new file (e.g., polygon.go) implementing market.Adapter
//  2. Register it in registry.go via RegisterAdapter()
//  3. Add it to the appropriate FallbackChain in the market package
package adapters

import "quantflow/internal/market"

// Ensure all built-in adapters implement market.Adapter.
var (
	_ market.Adapter = (*YahooAdapter)(nil)
	_ market.Adapter = (*EastMoneyAdapter)(nil)
	_ market.Adapter = (*BinanceAdapter)(nil)
	_ market.Adapter = (*AKShareAdapter)(nil)
	_ market.Adapter = (*TuShareAdapter)(nil)
	_ market.Adapter = (*TencentAdapter)(nil)
	_ market.Adapter = (*MootdxAdapter)(nil)
	_ market.Adapter = (*SinaAdapter)(nil)
	_ market.Adapter = (*BaiduAdapter)(nil)
	_ market.Adapter = (*OKXAdapter)(nil)
	_ market.Adapter = (*CoinGeckoAdapter)(nil)
	_ market.Adapter = (*PolygonAdapter)(nil)
)
