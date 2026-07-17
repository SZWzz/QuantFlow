export interface ApiKeyEntry {
  id: string
  name: string
  type: 'data_source' | 'broker' | 'ai'
  market?: string
  description: string
  keys: string[]
  verifyEndpoint: boolean
}

export const API_KEY_REGISTRY: ApiKeyEntry[] = [
  { id: 'alpaca', name: 'Alpaca Markets', type: 'broker', market: 'US', description: '美股实时行情 + 零佣金交易', keys: ['API Key', 'Secret Key'], verifyEndpoint: true },
  { id: 'binance', name: 'Binance', type: 'broker', market: 'CRYPTO', description: '加密货币现货 + 合约交易', keys: ['API Key', 'Secret Key'], verifyEndpoint: true },
  { id: 'okx', name: 'OKX', type: 'broker', market: 'CRYPTO', description: '加密货币交易', keys: ['API Key', 'Secret Key', 'Passphrase'], verifyEndpoint: true },
  { id: 'futu', name: '富途牛牛', type: 'broker', market: 'HK', description: '港股/A股行情 + 交易', keys: ['API Key', 'Secret Key'], verifyEndpoint: true },
  { id: 'ibkr', name: 'Interactive Brokers', type: 'broker', market: 'US', description: '全球多市场交易', keys: ['Account ID', 'API Key'], verifyEndpoint: true },
  { id: 'polygon', name: 'Polygon.io', type: 'data_source', market: 'US', description: '美股实时/历史行情', keys: ['API Key'], verifyEndpoint: true },
  { id: 'yahoo', name: 'Yahoo Finance', type: 'data_source', market: 'US', description: '免费美股数据（无需 API Key）', keys: [], verifyEndpoint: false },
  { id: 'eastmoney', name: '东方财富', type: 'data_source', market: 'CN', description: 'A股行情/财报/龙虎榜', keys: [], verifyEndpoint: false },
  { id: 'akshare', name: 'AKShare', type: 'data_source', market: 'CN', description: '免费开源 A股数据', keys: [], verifyEndpoint: false },
  { id: 'tushare', name: 'Tushare', type: 'data_source', market: 'CN', description: 'A股历史数据', keys: ['Token'], verifyEndpoint: true },
  { id: 'openai', name: 'OpenAI', type: 'ai', description: 'GPT-4o / o1 推理', keys: ['API Key', 'Organization ID'], verifyEndpoint: true },
  { id: 'ollama', name: 'Ollama', type: 'ai', description: '本地 LLM 推理', keys: [], verifyEndpoint: false },
  { id: 'deepseek', name: 'DeepSeek', type: 'ai', description: '国产大模型', keys: ['API Key'], verifyEndpoint: true },
]

export function filterByMarket(market: string): ApiKeyEntry[] {
  return API_KEY_REGISTRY.filter(e => e.market === market || !e.market)
}
