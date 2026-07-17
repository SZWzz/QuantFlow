export interface WorkflowTemplate {
  id: string
  name: string
  description: string
  category: 'starter' | 'trend' | 'mean_reversion' | 'crypto' | 'risk' | 'ai'
  nodes: Array<{ type: string; position: { x: number; y: number }; params?: Record<string, any> }>
  edges: Array<{ source: string; target: string; sourcePort: string; targetPort: string }>
}

export const WORKFLOW_TEMPLATES: WorkflowTemplate[] = [
  {
    id: 'moving-average-cross',
    name: '双均线交叉策略',
    description: '经典趋势跟踪：5日线上穿20日线买入，下穿卖出。适用于A股/美股日线。',
    category: 'trend',
    nodes: [
      { type: 'data_source', position: { x: 100, y: 200 }, params: { symbol: '600519.SH', interval: '1d' } },
      { type: 'sma', position: { x: 350, y: 100 }, params: { period: 5 } },
      { type: 'sma', position: { x: 350, y: 300 }, params: { period: 20 } },
      { type: 'cross_signal', position: { x: 600, y: 200 } },
      { type: 'order', position: { x: 850, y: 200 }, params: { quantity: 100, mode: 'paper' } },
    ],
    edges: [
      { source: 'data_source', target: 'sma5', sourcePort: 'close', targetPort: 'price' },
      { source: 'data_source', target: 'sma20', sourcePort: 'close', targetPort: 'price' },
      { source: 'sma5', target: 'cross_signal', sourcePort: 'ma', targetPort: 'fast' },
      { source: 'sma20', target: 'cross_signal', sourcePort: 'ma', targetPort: 'slow' },
      { source: 'cross_signal', target: 'order', sourcePort: 'signal', targetPort: 'input' },
    ],
  },
  {
    id: 'bollinger-breakout',
    name: '布林带突破策略',
    description: '价格突破上轨做多，跌破下轨做空。适合高波动性市场。',
    category: 'mean_reversion',
    nodes: [
      { type: 'data_source', position: { x: 100, y: 200 }, params: { symbol: 'AAPL', interval: '1d' } },
      { type: 'bollinger', position: { x: 350, y: 200 }, params: { period: 20, stddev: 2 } },
      { type: 'breakout_signal', position: { x: 600, y: 200 } },
      { type: 'order', position: { x: 850, y: 200 }, params: { quantity: 50, mode: 'paper' } },
    ],
    edges: [
      { source: 'data_source', target: 'bollinger', sourcePort: 'close', targetPort: 'price' },
      { source: 'bollinger', target: 'breakout_signal', sourcePort: 'bands', targetPort: 'bands' },
      { source: 'breakout_signal', target: 'order', sourcePort: 'signal', targetPort: 'input' },
    ],
  },
  {
    id: 'rsi-mean-reversion',
    name: 'RSI 均值回归策略',
    description: 'RSI<30 超卖买入，RSI>70 超买卖出。经典均值回归策略。',
    category: 'mean_reversion',
    nodes: [
      { type: 'data_source', position: { x: 100, y: 200 }, params: { symbol: 'TSLA', interval: '1d' } },
      { type: 'rsi', position: { x: 350, y: 200 }, params: { period: 14 } },
      { type: 'threshold_signal', position: { x: 600, y: 200 }, params: { buy_threshold: 30, sell_threshold: 70 } },
      { type: 'order', position: { x: 850, y: 200 }, params: { quantity: 100, mode: 'paper' } },
    ],
    edges: [
      { source: 'data_source', target: 'rsi', sourcePort: 'close', targetPort: 'price' },
      { source: 'rsi', target: 'threshold_signal', sourcePort: 'rsi', targetPort: 'value' },
      { source: 'threshold_signal', target: 'order', sourcePort: 'signal', targetPort: 'input' },
    ],
  },
  {
    id: 'crypto-arbitrage',
    name: '加密货币期现套利',
    description: '监控永续合约资金费率，正费率做多现货做空合约套利。',
    category: 'crypto',
    nodes: [
      { type: 'funding_rate', position: { x: 100, y: 200 }, params: { symbols: ['BTCUSDT', 'ETHUSDT'] } },
      { type: 'arb_signal', position: { x: 350, y: 200 }, params: { min_rate: 0.01 } },
      { type: 'order', position: { x: 600, y: 100 }, params: { side: 'buy', mode: 'live', market: 'CRYPTO' } },
      { type: 'order', position: { x: 600, y: 300 }, params: { side: 'sell', mode: 'live', market: 'CRYPTO' } },
    ],
    edges: [
      { source: 'funding_rate', target: 'arb_signal', sourcePort: 'rates', targetPort: 'rates' },
      { source: 'arb_signal', target: 'order_spot', sourcePort: 'spot', targetPort: 'input' },
      { source: 'arb_signal', target: 'order_perp', sourcePort: 'perp', targetPort: 'input' },
    ],
  },
  {
    id: 'portfolio-rebalance',
    name: '组合再平衡',
    description: '60/40 股债平衡，每月检查偏离度超过5%自动调仓。',
    category: 'risk',
    nodes: [
      { type: 'portfolio_snapshot', position: { x: 100, y: 200 } },
      { type: 'rebalance_calc', position: { x: 350, y: 200 }, params: { target_weights: { stocks: 0.6, bonds: 0.4 }, drift: 0.05 } },
      { type: 'order', position: { x: 600, y: 200 }, params: { mode: 'paper' } },
    ],
    edges: [
      { source: 'portfolio_snapshot', target: 'rebalance_calc', sourcePort: 'positions', targetPort: 'positions' },
      { source: 'rebalance_calc', target: 'order', sourcePort: 'trades', targetPort: 'input' },
    ],
  },
  {
    id: 'ai-sentiment',
    name: 'AI 情绪驱动策略',
    description: 'LLM 分析新闻情绪，正面买入负面卖出。适合事件驱动型交易。',
    category: 'ai',
    nodes: [
      { type: 'news_feed', position: { x: 100, y: 200 }, params: { symbol: 'AAPL', limit: 10 } },
      { type: 'ai_sentiment', position: { x: 350, y: 200 }, params: { provider: 'openai', model: 'gpt-4o' } },
      { type: 'order', position: { x: 600, y: 200 }, params: { mode: 'paper' } },
    ],
    edges: [
      { source: 'news_feed', target: 'ai_sentiment', sourcePort: 'articles', targetPort: 'text' },
      { source: 'ai_sentiment', target: 'order', sourcePort: 'signal', targetPort: 'input' },
    ],
  },
]
