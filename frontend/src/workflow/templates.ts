// Pre-built workflow templates for common quant strategies
export interface TemplateDef {
  id: string; name: string; description: string; icon: string
  nodes: { node_type: string; params: Record<string, any>; x: number; y: number }[]
  edges: { from: number; from_port: string; to: number; to_port: string }[]
}

const T = (type: string, x: number, y: number, params: Record<string, any> = {}) => ({ node_type: type, params, x, y })
const E = (from: number, fromPort: string, to: number, toPort: string) => ({ from, from_port: fromPort, to, to_port: toPort })

export const TEMPLATES: TemplateDef[] = [
  {
    id: 'golden-cross', name: '金叉选股', description: '5日均线上穿20日均线产生买入信号', icon: '📈',
    nodes: [
      T('data_loader', 100, 100, { symbol: '600519' }),
      T('sma', 350, 60, { period: 5 }), T('sma', 350, 180, { period: 20 }),
      T('cross_over', 600, 100, {}), T('entry_signal', 850, 100, {}),
    ],
    edges: [E(0,'ohlcv',1,'input'),E(0,'ohlcv',2,'input'),E(1,'output',3,'fast'),E(2,'output',3,'slow'),E(3,'signal',4,'signal')],
  },
  {
    id: 'macd-divergence', name: 'MACD 底背离', description: '价格新低但 MACD DIF 不创新低 → 买入', icon: '📉',
    nodes: [
      T('data_loader', 100, 100, { symbol: '000001' }),
      T('macd', 350, 60, {}), T('rolling_maxmin', 350, 200, { period: 20, mode: 'min' }),
      T('compare', 600, 120, { op: 'divergence' }), T('entry_signal', 850, 120, {}),
    ],
    edges: [E(0,'ohlcv',1,'input'),E(0,'ohlcv',2,'input'),E(1,'dif',3,'fast'),E(2,'output',3,'slow'),E(3,'result',4,'signal')],
  },
  {
    id: 'multi-factor', name: '多因子打分', description: '动量+波动+成交量三因子等权打分排名', icon: '📊',
    nodes: [
      T('data_loader', 100, 100, { symbol: '000300' }),
      T('pct_change', 350, 30, { period: 20 }), T('std_dev', 350, 140, { period: 20 }), T('rolling_zscore', 350, 250, { period: 20 }),
      T('scale', 600, 30, {}), T('scale', 600, 140, {}), T('scale', 600, 250, {}),
      T('merge', 850, 140, { method: 'weighted', weights: [0.4, 0.3, 0.3] }),
      T('rank_select', 1100, 140, { top_n: 20 }),
    ],
    edges: [
      E(0,'ohlcv',1,'input'),E(0,'ohlcv',2,'input'),E(0,'ohlcv',3,'input'),
      E(1,'output',4,'input'),E(2,'output',5,'input'),E(3,'output',6,'input'),
      E(4,'output',7,'a'),E(5,'output',7,'b'),E(6,'output',7,'c'),E(7,'output',8,'input'),
    ],
  },
  {
    id: 'bollinger-break', name: '布林带突破', description: '突破下轨买入 + 上轨止盈', icon: '💰',
    nodes: [
      T('data_loader', 100, 100, { symbol: '300750' }), T('bollinger', 350, 100, {}),
      T('compare', 600, 30, { op: 'lt' }), T('compare', 600, 180, { op: 'gt' }),
      T('entry_signal', 850, 30, {}), T('exit_signal', 850, 180, {}),
    ],
    edges: [E(0,'ohlcv',1,'input'),E(1,'lower',2,'fast'),E(1,'middle',2,'slow'),E(1,'upper',3,'fast'),E(1,'middle',3,'slow'),E(2,'result',4,'signal'),E(3,'result',5,'signal')],
  },
  {
    id: 'rsi-oversold', name: 'RSI 超卖反弹', description: 'RSI<30 买入 + RSI>70 卖出', icon: '⚡',
    nodes: [
      T('data_loader', 100, 100, { symbol: '601318' }), T('rsi', 350, 100, { period: 14 }),
      T('threshold_signal', 600, 40, { threshold: 30, direction: 'below' }),
      T('threshold_signal', 600, 180, { threshold: 70, direction: 'above' }),
      T('entry_signal', 850, 40, {}), T('exit_signal', 850, 180, {}),
    ],
    edges: [E(0,'ohlcv',1,'input'),E(1,'output',2,'input'),E(1,'output',3,'input'),E(2,'signal',4,'signal'),E(3,'signal',5,'signal')],
  },
  {
    id: 'pair-trading', name: '均值回归配对', description: '价差突破 2σ → 做多/做空', icon: '🛡️',
    nodes: [
      T('data_loader', 100, 80, { symbol: '600519' }), T('data_loader', 100, 220, { symbol: '000858' }),
      T('arithmetic', 350, 140, { op: 'subtract' }),
      T('std_dev', 600, 50, { period: 60 }), T('rolling_zscore', 600, 160, { period: 60 }),
      T('threshold_signal', 850, 50, { threshold: 2, direction: 'above' }),
      T('threshold_signal', 850, 160, { threshold: -2, direction: 'below' }),
      T('entry_signal', 1100, 50, {}), T('entry_signal', 1100, 160, {}),
    ],
    edges: [E(0,'ohlcv',2,'a'),E(1,'ohlcv',2,'b'),E(2,'output',3,'input'),E(2,'output',4,'input'),E(4,'output',6,'input'),E(3,'output',5,'input'),E(5,'signal',7,'signal'),E(6,'signal',8,'signal')],
  },
  {
    id: 'macd-rsi-combo', name: 'MACD+RSI 共振', description: 'MACD 金叉 + RSI<50 → 过滤假信号', icon: '🧠',
    nodes: [
      T('data_loader', 100, 100, { symbol: '000001' }),
      T('macd', 350, 30, {}), T('rsi', 350, 200, { period: 14 }),
      T('cross_over', 600, 30, {}), T('threshold_signal', 600, 200, { threshold: 50, direction: 'below' }),
      T('signal_combine', 850, 110, { op: 'and' }), T('entry_signal', 1100, 110, {}),
    ],
    edges: [E(0,'ohlcv',1,'input'),E(0,'ohlcv',2,'input'),E(1,'dif',3,'fast'),E(1,'dea',3,'slow'),E(2,'output',4,'input'),E(3,'signal',5,'a'),E(4,'signal',5,'b'),E(5,'output',6,'signal')],
  },
]
