import { vi } from 'vitest'
import { config } from '@vue/test-utils'
import { mockWailsIPC, mockWebSocket, mockI18n } from './src/__tests__/mocks'

mockWailsIPC()
mockWebSocket()
mockI18n()

const originalSetTimeout = global.setTimeout
global.setTimeout = ((fn: any, ms: any, ...args: any[]) => {
  if (typeof fn === 'string' && fn.includes('window')) return 0
  if (fn.toString().includes('window')) return 0
  return originalSetTimeout(fn, ms, ...args)
}) as any

global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any

HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
  clearRect: vi.fn(),
  fillRect: vi.fn(),
  getImageData: vi.fn(() => ({ data: [] })),
  putImageData: vi.fn(),
  createImageData: vi.fn(() => []),
  setTransform: vi.fn(),
  drawImage: vi.fn(),
  save: vi.fn(),
  fillText: vi.fn(),
  restore: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  closePath: vi.fn(),
  stroke: vi.fn(),
  fill: vi.fn(),
  arc: vi.fn(),
  canvas: { width: 0, height: 0 },
})) as any

config.global.mocks.$t = (key: string) => {
  const map: Record<string, string> = {
    'broker.title': 'Broker Status',
    'broker.refresh': 'Refresh',
    'broker.refreshing': 'Refreshing...',
    'broker.notConfigured': 'Not Configured',
    'broker.testConnection': 'Test Connection',
    'broker.paperTrading': 'Paper Trading',
    'broker.market_label': 'Market',
    'broker.no_brokers': 'No brokers configured',
    'common.refresh': 'Refresh',
    'common.summary': 'Summary',
    'common.connected': 'Connected',
    'common.disconnected': 'Disconnected',
    'basketOrder.title': 'Basket Order',
    'basketOrder.execute': 'Execute Basket',
    'basketOrder.addRow': 'Add Row',
    'basketOrder.importCSV': 'Import CSV',
    'execution.title': 'Execution History',
    'geopolitics.panelTitle': '地缘政治风险',
    'geopolitics.noEvents': '暂无事件',
    'govData.panelTitle': 'Government Data',
    'govData.source1': 'Congress',
    'govData.source2': 'SEC',
    'govData.source3': 'Treasury',
    'predictionMarket.panelTitle': '预测市场',
    'predictionMarket.noEvents': '暂无事件',
    'satellite.panelTitle': 'Satellite Monitoring',
    'satellite.noData': 'No satellite data',
    'sentiment.panelTitle': 'Market Sentiment',
    'sentiment.noData': 'Sentiment data unavailable',
    'distribution.title': 'Return Distribution',
    'distribution.calculate': '计算',
    'distribution.placeholder': 'Enter a symbol and click 计算',
    'monteCarlo.title': 'Monte Carlo',
    'monteCarlo.run': 'Run',
    'monteCarlo.placeholder': 'Run simulation to see results',
    'equityCurve.title': 'Equity Curve',
    'equityCurve.placeholder': 'No equity curve data',
    'rebalance.title': 'Rebalance',
    'rebalance.placeholder': 'No rebalance data',
    'correlation.title': 'Correlation',
    'correlation.compute': 'Compute',
    'correlation.placeholder': 'Select assets to compute',
    'surfaceChart.title': 'Volatility Surface',
    'surfaceChart.placeholder': 'Select symbol to view surface',
    'cryptoOverview.title': 'Crypto Overview',
    'cryptoOverview.btcDominance': 'BTC Dominance',
    'tickerTape.title': 'Ticker Tape',
    'heatmap.title': 'Heatmap',
    'heatmap.noData': 'No heatmap data',
  }
  return map[key] || key.split('.').pop() || key
}
