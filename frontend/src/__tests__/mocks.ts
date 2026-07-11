import { vi } from 'vitest'

function _toTitle(str: string): string {
  return str
    .replace(/[_-]/g, ' ')
    .replace(/([A-Z])/g, ' $1')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/\b[a-z]/g, c => c.toUpperCase())
}

const t = (key: string) => {
  const map: Record<string, string> = {
    // Broker
    'broker.paperTrading': 'Paper Trading',
    'broker.notConfigured': 'Not Configured',
    'broker.testConnection': 'Test Connection',
    // Common
    'common.refresh': 'Refresh',
    'common.summary': 'Summary',
    'common.connected': 'Connected',
    'common.disconnected': 'Disconnected',
    'common.loading': 'Loading',
    'common.no_data': 'No data',
    'common.search': 'Search',
    'common.symbol': 'Symbol',
    'common.time': 'Time',
    'common.price': 'Price',
    'common.amount': 'Amount',
    'common.total': 'Total',
    'common.date': 'Date',
    'common.status': 'Status',
    'common.type': 'Type',
    'common.name': 'Name',
    'common.save': 'Save',
    'common.close': 'Close',
    'common.select': 'Select',
    'common.testing': 'Testing...',
    'common.panel_error': 'Panel error',
    'common.size': 'Size',
    // Action Center
    'actionCenter.title': 'Action Center',
    'actionCenter.dismiss': 'Dismiss',
    'actionCenter.approve': 'Approve',
    // Basket Order
    'basketOrder.title': 'Basket Order',
    'basketOrder.execute': 'Execute Basket',
    'basketOrder.addRow': 'Add Row',
    'basketOrder.importCSV': 'Import CSV',
    // Correlation
    'correlation.compute': 'Compute',
    // Crypto Overview
    'cryptoOverview.title': 'Crypto Overview',
    'cryptoOverview.btcDominance': 'BTC Dominance',
    // Distribution
    'distribution.calculate': 'Calculate',
    // Execution
    'execution.title': 'Execution',
    // Geopolitics
    'geopolitics.title': 'Geopolitics Risk',
    'geopolitics.panelTitle': '地缘政治风险',
    'geopolitics.noEvents': '暂无事件',
    // Gov Data
    'govData.title': 'Government Data',
    'govData.source1': 'Congress',
    'govData.source2': 'SEC',
    'govData.source3': 'Treasury',
    // Heatmap
    'heatmap.title': 'Heatmap',
    // Market
    'misc.liquidation': 'Liquidation',
    'misc.short_interest': 'Short Interest',
    'misc.days_to_cover': 'Days to Cover',
    'misc.direction': 'Direction',
    'misc.symbol_filter': 'Filter symbol',
    'misc.liquidation_total': 'Liquidation Total',
    'misc.max_single': 'Max Single',
    'misc.long_liq': 'Long Liq',
    'misc.short_liq': 'Short Liq',
    'misc.short_pct': 'Short %',
    'misc.trend': 'Trend',
    'misc.csv_export': 'Export CSV',
    // ML
    'ml.factor_analysis': 'Factor Analysis',
    'ml.models': 'Models',
    'ml.rl_monitor': 'RL Monitor',
    // Monte Carlo
    'monteCarlo.title': 'Monte Carlo Simulation',
    'monteCarlo.run': 'Run',
    'monteCarlo.placeholder': 'Run simulation to see results',
    // Equity Curve
    'equityCurve.title': 'Equity Curve',
    'equityCurve.placeholder': 'No equity curve data',
    // Rebalance
    'rebalance.title': 'Rebalance',
    'rebalance.placeholder': 'No rebalance data',
    // Prediction Market
    'predictionMarket.title': 'Prediction Market',
    'predictionMarket.panelTitle': '预测市场',
    'predictionMarket.noEvents': '暂无事件',
    // Portfolio
    'portfolio.total_pnl': 'Total PnL',
    'portfolio.position_count': 'Positions',
    'portfolio.no_positions': 'No positions',
    'portfolio.avg_price': 'Avg Price',
    'portfolio.market_price': 'Market Price',
    'portfolio.market_value': 'Market Value',
    'portfolio.pnl': 'PnL',
    'portfolio.alloc': 'Allocation',
    'portfolio.quantity': 'Quantity',
    // Quote
    'quote.symbol': 'Symbol',
    // Satellite
    'satellite.title': 'Satellite Monitoring',
    'satellite.solar_radiation': 'Solar Radiation',
    'satellite.energy_kwh': 'kWh',
    'satellite.wind_speed': 'Wind Speed',
    'satellite.fire_count': 'Fire Count',
    'satellite.linked_assets': 'Linked Assets',
    'satellite.no_data': 'No satellite data',
    'satellite.loading_chart': 'Loading chart...',
    'satellite.no_history': 'No history data',
    'satellite.stable_indicator': 'Stable',
    // Sentiment
    'sentiment.title': 'Market Sentiment',
    'sentiment.noData': 'Sentiment data unavailable',
    // Settings
    'settings.llm_providers': 'LLM Providers',
    'settings.llm_custom_name': 'Custom Name',
    'settings.llm_openai_key': 'API Key',
    'settings.llm_openai_url': 'API URL',
    'settings.llm_custom_models': 'Model List',
    'settings.llm_load_models': 'Load Models',
    'settings.llm_test': 'Test Connection',
    'settings.llm_apply_models': 'Apply Models',
    'settings.llm_default_model': 'Default Model',
    'settings.llm_refresh_all': 'Refresh All',
    'settings.llm_no_models': 'No models available',
    'settings.llm_models_loaded': '{count} models loaded',
    'settings.llm_save_hint': 'Changes saved',
    'settings.llm_test_success': 'Connection successful',
    'settings.llm_test_fail': 'Connection failed',
    // Surface Chart
    'surfaceChart.title': 'Volatility Surface',
    'surfaceChart.placeholder': 'Select symbol to view surface',
    // Ticker Tape
    'tickerTape.title': 'Ticker Tape',
    // Trade
    'trade.side': 'Side',
    'trade.quantity': 'Quantity',
    'trade.order_id': 'Order ID',
    'trade.buy': 'Buy',
    'trade.sell': 'Sell',
    'trade.filled_pct': 'Filled %',
    'trade.cancel_order': 'Cancel Order',
    'trade.no_orders': 'No orders',
    'trade.today_orders': "Today's Orders",
    'trade.total_value': 'Total Value',
    'trade.all_status': 'All Status',
    // Watchlist
    'watchlist.ticker_tape': 'Ticker Tape',
    // Workflow
    'workflow.action_center': 'Action Center',
    'workflow.no_recent_trades': 'No recent trades',
    'workflow.no_trades': 'No trades',
    'workflow.fee': 'Fee',
    'workflow.load_more': 'Load More',
    'workflow.no_executions': 'No executions',
    'workflow.add_to_workflow': 'Add to Workflow',
    // Geo
    'geo.title': '地缘政治风险',
    'geo.neutral_signal': 'Neutral',
    'geo.risk_level': 'Risk Level',
    'geo.sentiment_score': 'Sentiment Score',
    'geo.sentiment_change': 'Sentiment Change',
    'geo.discussion_change': 'Discussion Change',
    'geo.no_sentiment': 'No sentiment data',
    'geo.linked_assets': 'Linked Assets',
    // Prediction Market
    'prediction.title': '预测市场',
    // Distribution
    'distribution.title': 'Return Distribution',
    'distribution.placeholder': 'Enter a symbol and click Compute',
    // Correlation
    'correlation.title': 'Correlation Matrix',
    'correlation.placeholder': 'Enter symbols and click Compute',
    // Research
    'research.sentiment': 'Sentiment Analysis',
    'research.hint_enter_symbol': 'Enter a symbol and press ↵',
    'research.keywords': 'Keywords',
    'research.no_keywords': 'No keywords',
    // Misc
    'misc.volatility_surface': 'Volatility Surface',
    'misc.heatmap': 'Heatmap',
    'misc.no_hk_sector_data': 'No HK sector data',
    // Panels
    'panels.ipo_calendar': 'IPO Calendar',
    'panels.today_apply': 'Today Apply',
    'panels.upcoming': 'Upcoming',
    'panels.recent': 'Recent',
    'panels.no_data': 'No data available',
    // Monitor
    'monitor.go_runtime': 'Go Runtime',
    'monitor.goroutines': 'Goroutines',
    'monitor.heap_memory': 'Heap Memory',
    'monitor.system_memory': 'System Memory',
    'monitor.uptime': 'Uptime',
    'monitor.workflow_engine': 'Workflow Engine',
    'monitor.registered_nodes': 'Registered Nodes',
    'monitor.cache_size': 'Cache Size',
    'monitor.active_runs': 'Active Runs',
    // Broker
    'broker.title': 'Broker Status',
    'broker.refresh': 'Refresh',
    'broker.refreshing': 'Refreshing...',
    'broker.market_label': 'Market',
    'broker.no_brokers': 'No brokers configured',
    // Watchlist
    'watchlist.title': 'Watchlist',
    'watchlist.empty': 'No symbols',
    'watchlist.column_settings': 'Column Settings',
    'watchlist.polling_paused': 'Polling Paused',
    'watchlist.context_open_kline': 'Open K-line',
    'watchlist.context_copy': 'Copy Code',
    'watchlist.context_delete': 'Delete',
    // Common additions
    'common.delete': 'Delete',
    'common.retry': 'Retry',
  }
  return map[key] || _toTitle(key.split('.').pop() || key)
}

export { t }

export function mockWailsIPC() {
  const app = {
    SearchSymbols: vi.fn().mockResolvedValue({ data: [] }),
    GetQuote: vi.fn().mockImplementation((market: string, symbol: string) => {
      return {
        symbol, last: 150.0, change: 1.5, changePct: 1.0, open: 148.0,
        high: 152.0, low: 147.5, volume: 1000000, turnover: 150000000,
        bid: 149.9, ask: 150.1, prevClose: 148.5,
      }
    }),
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
