export interface PanelToNodeEntry {
  nodeType: string
  label: string
  multi?: string[]
}

export const PANEL_TO_NODE: Record<string, PanelToNodeEntry> = {
  'candlestick':         { nodeType: 'data_loader',   label: 'Data Loader' },
  'watchlist':           { nodeType: 'loop',          label: 'Loop' },
  'indicator':           { nodeType: 'sma',           label: 'SMA',
                             multi: ['macd', 'rsi', 'bollinger', 'ema'] },
  'stock-scanner':       { nodeType: 'rank_select',   label: 'Rank Select' },
  'factor-analysis':     { nodeType: 'factor',        label: 'Factor' },
  'backtest':            { nodeType: 'backtest',      label: 'Backtest' },
  'sentiment':           { nodeType: 'sentiment',     label: 'Sentiment' },
  'news':                { nodeType: 'news_fetcher',  label: 'News Fetcher' },
  'correlation':         { nodeType: 'math_op',       label: 'Math Op' },
  'distribution':        { nodeType: 'math_op',       label: 'Math Op' },
  'financials':          { nodeType: 'financials',    label: 'Financials' },
  'peer-comparison':     { nodeType: 'peer_compare',  label: 'Peer Compare' },
  'analyst-estimates':   { nodeType: 'analyst_estimates', label: 'Analyst Estimates' },
  'insider-trading':     { nodeType: 'insider_trades',label: 'Insider Trades' },
  'prediction-market':   { nodeType: 'prediction_market', label: 'Prediction Market' },
  'geopolitics':         { nodeType: 'geopolitics',   label: 'Geopolitics' },
  'satellite':           { nodeType: 'satellite',     label: 'Satellite' },
  'macro':               { nodeType: 'gov_data',      label: 'Gov Data' },
  'fundflow':            { nodeType: 'data_loader',   label: 'Data Loader' },
  'market-overview':     { nodeType: 'data_loader',   label: 'Data Loader' },
}

export function getPanelToNode(panelId: string): PanelToNodeEntry | undefined {
  return PANEL_TO_NODE[panelId]
}
