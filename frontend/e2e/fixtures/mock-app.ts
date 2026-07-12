import type { Page } from '@playwright/test'

export async function setupMocks(page: Page) {
  await page.addInitScript(() => {
    // Mock Wails IPC — window.go.main.App
    const mockApp = {
      SearchSymbols: async (q: string, limit: number) => {
        const symbols = [
          { code: '600519', name: '贵州茅台', market: 'CN', pinyin: 'gzmt' },
          { code: '000001', name: '平安银行', market: 'CN', pinyin: 'payh' },
          { code: '300750', name: '宁德时代', market: 'CN', pinyin: 'ndsd' },
          { code: 'AAPL', name: 'Apple Inc.', market: 'US', pinyin: '' },
          { code: '00700', name: '腾讯控股', market: 'HK', pinyin: '' },
        ]
        const filtered = q ? symbols.filter(s => s.code.includes(q) || s.name.includes(q)) : symbols
        return { data: filtered.slice(0, limit) }
      },

      GetQuote: async (market: string, symbol: string) => {
        const names: Record<string, string> = {
          '600519': '贵州茅台', '000001': '平安银行', '300750': '宁德时代',
          'AAPL': 'Apple Inc.', '00700': '腾讯控股',
        }
        const prices: Record<string, number> = {
          '600519': 1650.00, '000001': 12.50, '300750': 210.00,
          'AAPL': 195.50, '00700': 320.00,
        }
        return {
          data: [{
            symbol, name: names[symbol] || symbol,
            last: prices[symbol] || 100, change: 1.5, changePct: 1.0,
            open: (prices[symbol] || 100) - 2, high: (prices[symbol] || 100) + 2,
            low: (prices[symbol] || 100) - 3, volume: 1000000, turnover: 150000000,
            bid: (prices[symbol] || 100) - 0.1, ask: (prices[symbol] || 100) + 0.1,
            prevClose: (prices[symbol] || 100) - 1.5, timestamp: Date.now(),
          }]
        }
      },

      FetchOHLCV: async () => ({ data: [] }),
      GetMinuteLine: async () => ({ data: [] }),

      PlaceOrder: async (_symbol: string, _side: string, _orderType: string, _brokerName: string, _qty: number, _price: number) => {
        return { id: 'test-' + Date.now(), symbol: _symbol, side: _side, orderType: _orderType, status: 'filled', quantity: _qty, price: _price, placedAt: new Date().toISOString() }
      },

      GetPositions: async () => {
        return [
          { symbol: '600519', name: '贵州茅台', quantity: 100, avgPrice: 1600, marketPrice: 1650, pnl: 5000, pnlPct: 3.13, market: 'CN', allocPct: 30 },
          { symbol: 'AAPL', name: 'Apple Inc.', quantity: 50, avgPrice: 180, marketPrice: 195.5, pnl: 775, pnlPct: 8.61, market: 'US', allocPct: 20 },
        ]
      },

      GetOrders: async () => {
        return [
          { id: 'order-001', symbol: '600519', side: 'buy', orderType: 'limit', quantity: 100, price: 1600, filledQty: 100, status: 'filled', placedAt: new Date().toISOString() },
          { id: 'order-002', symbol: 'AAPL', side: 'buy', orderType: 'market', quantity: 50, price: 0, filledQty: 50, status: 'filled', placedAt: new Date().toISOString() },
        ]
      },

      GetTrades: async () => {
        return [
          { id: 'trade-001', orderId: 'order-001', symbol: '600519', side: 'buy', quantity: 100, price: 1600, timestamp: new Date().toISOString() },
          { id: 'trade-002', orderId: 'order-002', symbol: 'AAPL', side: 'buy', quantity: 50, price: 195.5, timestamp: new Date().toISOString() },
        ]
      },

      GetPortfolioSummary: async () => {
        return { totalValue: 250000, cashBalance: 50000, marketValue: 200000, totalPnL: 5775, totalPnLPct: 2.36 }
      },

      GetPortfolioAllocation: async () => {
        return { byMarket: { CN: 70, US: 30 }, bySector: {}, byCurrency: {} }
      },

      GetEquityCurve: async () => {
        return [
          { date: '2026-01-01', nav: 100000, benchmark: 100000 },
          { date: '2026-06-01', nav: 105775, benchmark: 103000 },
        ]
      },

      ListBacktestHistory: async () => {
        return {
          data: [
            { id: 1, name: '双均线策略', symbols: '600519', startDate: '2025-01-01', endDate: '2026-01-01', totalReturn: 15.5, sharpe: 1.2, maxDrawdown: -8.5, createdAt: new Date().toISOString() },
            { id: 2, name: '动量突破策略', symbols: 'AAPL', startDate: '2025-06-01', endDate: '2026-06-01', totalReturn: 22.3, sharpe: 1.8, maxDrawdown: -12.0, createdAt: new Date().toISOString() },
          ]
        }
      },

      GetStoredBacktestResult: async (_ctx: any, id: number) => {
        return {
          id, name: '双均线策略', symbols: '600519',
          configJson: '{}', tradesJson: '[]', equityCurveJson: '[]', ohlcvDataJson: '[]',
          metricsJson: JSON.stringify({ totalReturn: 15.5, cagr: 15.5, maxDrawdown: -8.5, sharpe: 1.2, sortino: 1.5, calmar: 1.8, winRate: 55, profitFactor: 1.6, tradeCount: 45 }),
          startDate: '2025-01-01', endDate: '2026-01-01', createdAt: new Date().toISOString(),
        }
      },

      DeleteBacktestResult: async () => true,
      ClearBacktestResults: async () => 2,

      GetBrokerStatuses: async () => {
        return [
          { name: 'paper', label: 'Paper Trading', market: '模拟', connected: true, detail: '本地模拟撮合' },
          { name: 'binance', label: 'Binance', market: '加密', connected: false, detail: '未配置' },
        ]
      },

      GetBrokerConnection: async () => ({ connected: false }),

      GetMarketOverview: async (_mkt: string) => {
        return {
          indices: [{ code: '000001.SH', name: '上证指数', price: 3000, change_pct: 0.5 }],
          breadth: { advancers: 1500, decliners: 500, unchanged: 200 },
          sentiment: { score: 55, label: '中性' },
        }
      },

      Chat: async () => 'Mock AI response',
      ListProfiles: async () => [{ name: 'general', display: '通用助理' }],
      GetConfig: async () => ({}),
      GetNotifications: async () => [],
      GetSystemStats: async () => ({ goRuntime: 'go1.25', goroutines: 42, heapMB: 64, systemMB: 256, uptime: '2h' }),
    }

    // Inject before Vue app mounts
    ;(window as any).go = { main: { App: mockApp } }

    // Mock Wails dialogs (polyfill from wails.ts expects these)
    ;(window as any).__wailsDialogs = {
      Question: async (opts: any) => opts?.Buttons?.find((b: any) => b.IsDefault)?.Label || '确定',
      Info: async () => {},
    }
  })

  // Mock WebSocket — no-op constructor
  await page.addInitScript(() => {
    const OrigWS = window.WebSocket
    ;(window as any).WebSocket = class MockWS extends OrigWS {
      constructor(url: string) {
        super(url)
        // Prevent actual connection attempts
      }
    }
  })
}
