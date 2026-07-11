import { vi } from 'vitest'

const t = (key: string) => {
  const map: Record<string, string> = {
    'broker.paperTrading': 'Paper Trading',
    'broker.notConfigured': 'Not Configured',
    'broker.testConnection': 'Test Connection',
    'common.refresh': 'Refresh',
    'common.summary': 'Summary',
    'common.connected': 'Connected',
    'common.disconnected': 'Disconnected',
    'distribution.placeholder': 'Enter a symbol and click Calculate',
    'distribution.calculate': 'Calculate',
    'actionCenter.title': 'Action Center',
    'actionCenter.dismiss': 'Dismiss',
    'actionCenter.approve': 'Approve',
    'basketOrder.title': 'Basket Order',
    'basketOrder.execute': 'Execute Basket',
    'basketOrder.addRow': 'Add Row',
    'basketOrder.importCSV': 'Import CSV',
    'correlation.placeholder': 'Select symbols and compute correlation',
    'correlation.compute': 'Compute',
    'cryptoOverview.title': 'Crypto Overview',
    'cryptoOverview.btcDominance': 'BTC Dominance',
    'execution.title': 'Execution',
    'heatmap.title': 'Heatmap',
    'geopolitics.title': 'Geopolitics Risk',
    'govData.title': 'Government Data',
    'predictionMarket.title': 'Prediction Market',
    'satellite.title': 'Satellite Monitoring',
    'sentiment.title': 'Market Sentiment',
    'surfaceChart.title': 'Volatility Surface',
    'monteCarlo.title': 'Monte Carlo Simulation',
    'monteCarlo.run': 'Run Simulation',
    'equityCurve.title': 'Equity Curve',
    'rebalance.title': 'Rebalance',
    'tickerTape.title': 'Ticker Tape',
  }
  return map[key] || key.split('.').pop() || key
}

export function mockWailsIPC() {
  const app = {
    SearchSymbols: vi.fn().mockResolvedValue({ data: [] }),
    GetQuote: vi.fn().mockResolvedValue(null),
    FetchOHLCV: vi.fn().mockResolvedValue([]),
    GetMinuteLine: vi.fn().mockResolvedValue([]),
    GetMarketOverview: vi.fn<[string?], any>().mockResolvedValue({
      indices: [{ code: '000001.SH', name: '上证指数', price: 3000, change_pct: 0.5 }],
      breadth: { advancers: 1500, decliners: 500 },
      sectors: [{ name: '科技', changePct: 1.2 }],
    }),
    GetIndustryRanks: vi.fn<[string, number?], any>().mockResolvedValue([
      { name: '科技', changePct: 2.5 },
      { name: '金融', changePct: 1.0 },
    ]),
    ListNodes: vi.fn().mockResolvedValue([]),
    GetNodePorts: vi.fn().mockResolvedValue({ inputs: [], outputs: [] }),
    GetAbnormalStocks: vi.fn().mockResolvedValue([]),
    GetRealtimeDepth: vi.fn().mockResolvedValue(null),
    GetFundingRate: vi.fn().mockResolvedValue([]),
    FetchBacktest: vi.fn().mockResolvedValue(null),
    GetPortfolioSummary: vi.fn().mockResolvedValue(null),
    GetOrders: vi.fn<[], any>().mockResolvedValue([
      { order_id: 'ord_001', symbol: '600519', side: 'buy', status: 'filled', quantity: 100, price: 1800 },
    ]),
    GetTrades: vi.fn<[], any>().mockResolvedValue([
      { trade_id: 'trd_001', symbol: '600519', side: 'buy', quantity: 100, price: 1800 },
    ]),
    GetEquityCurve: vi.fn<[], any>().mockResolvedValue(
      Array.from({ length: 252 }, (_, i) => ({
        date: `2024-${String(Math.floor(i / 30) + 1).padStart(2, '0')}-${String((i % 30) + 1).padStart(2, '0')}`,
        nav: 1 + i * 0.001 + Math.random() * 0.02,
        benchmark: 1 + i * 0.0008,
      }))
    ),
    CancelOrder: vi.fn().mockResolvedValue(true),
    GetBrokerStatus: vi.fn().mockResolvedValue([]),
    GetBrokerStatuses: vi.fn().mockResolvedValue([
      { name: 'Futu', label: 'Futu', connected: false, type: 'paper', market: 'HK', detail: '' },
      { name: 'Binance', label: 'Binance', connected: true, type: 'live', market: 'CRYPTO', detail: 'API connected' },
      { name: 'Alpaca', label: 'Alpaca', connected: false, type: 'paper', market: 'US', detail: '' },
      { name: 'IBKR', label: 'IBKR', connected: false, type: 'paper', market: 'US', detail: '' },
      { name: 'OKX', label: 'OKX', connected: false, type: 'paper', market: 'CRYPTO', detail: '' },
      { name: 'Paper Trading', label: 'Paper Trading', connected: true, type: 'paper', market: 'CN', detail: 'Active' },
    ]),
    GetEvents: vi.fn().mockResolvedValue([
      { id: 'evt_001', type: 'stop_loss', symbol: '600519', message: 'Stop-Loss Triggered', severity: 'warning', timestamp: Date.now() },
      { id: 'evt_002', type: 'large_order', symbol: 'AAPL', message: 'Large order: AAPL 10000 shares', severity: 'info', timestamp: Date.now() },
    ]),
    DismissEvent: vi.fn().mockResolvedValue(true),
    ApproveEvent: vi.fn().mockResolvedValue(true),
    GetReturnDistribution: vi.fn().mockResolvedValue({ returns: [], mean: 0, std: 0, skewness: 0, kurtosis: 0 }),
    ListBacktestHistory: vi.fn().mockResolvedValue([]),
    GetSystemStats: vi.fn().mockResolvedValue({ cpu: 0, memory: 0, uptime: 0 }),
    GetCryptoOverview: vi.fn().mockResolvedValue([
      { symbol: 'BTCUSDT', last: 50000, changePct: 2.5, volume: 1e9 },
      { symbol: 'ETHUSDT', last: 3000, changePct: 1.8, volume: 5e8 },
    ]),
    GetNews: vi.fn().mockResolvedValue([
      { id: 'n_001', title: 'Market News', source: 'Reuters', date: '2024-01-01' },
    ]),
  }
  ;(window as any).go = { main: { App: app } }
  return app
}

export function mockWebSocket() {
  class MockWebSocket {
    readyState = WebSocket.OPEN
    send = vi.fn()
    close = vi.fn()
    addEventListener = vi.fn()
    removeEventListener = vi.fn()
  }
  vi.stubGlobal('WebSocket', MockWebSocket as any)
}

export function mockI18n() {
  // Mock vue-i18n module for script-level useI18n().t()
  vi.mock('vue-i18n', async () => {
    const actual = await vi.importActual('vue-i18n')
    return {
      ...(actual as any),
      useI18n: () => ({
        t,
        locale: { value: 'en' },
        availableLocales: ['zh-CN', 'en'],
      }),
    }
  })
  // Register $t as a global Vue property for template usage
  // Tests must use mount options: global: { mocks: { $t: t } }
}
