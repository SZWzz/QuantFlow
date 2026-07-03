// Shared label map for all 85+ workflow node types.
// Used by both NodePalette (display) and CustomNode (canvas card header).
export const NODE_LABELS: Record<string, string> = {
  // ── Data ──
  data_loader: '数据加载',
  merge: '数据合并',
  resample: '重采样',
  filter: '过滤',

  // ── Indicators (standard) ──
  sma: '简单均线', ema: '指数均线', macd: 'MACD', rsi: 'RSI', bollinger: '布林带',

  // ── Indicators (TDX 通达信) ──
  indicator_kdj: 'KDJ', indicator_dmi: 'DMI', indicator_atr: 'ATR', indicator_wr: 'WR',
  indicator_cci: 'CCI', indicator_bias: 'BIAS', indicator_bias_signal: 'BIAS信号',
  indicator_obv: 'OBV', indicator_mfi: 'MFI', indicator_sar: 'SAR',
  indicator_vwap: 'VWAP', indicator_zhuoyao: '主力监控', indicator_aroon: 'AROON',
  indicator_asi: 'ASI', indicator_brar: 'BRAR', indicator_mass: 'MASS',
  indicator_psy: 'PSY', indicator_roc: 'ROC', indicator_bbi: 'BBI',

  // ── Signal ──
  cross_signal: '交叉信号', threshold_signal: '阈值信号', signal_combine: '信号组合',
  rank_select: '排名选择', hold_signal: '持仓信号',
  entry_signal: '入场信号', exit_signal: '离场信号',

  // ── Alpha ──
  factor: '因子', pct_change: '涨跌幅', delta: '差值', std_dev: '标准差',
  rank: '排名', scale: '标准化', cross_over: '上穿检测', compare: '比较',
  bool_combine: '布尔组合', rolling_maxmin: '滚动极值', rolling_zscore: '滚动Z值',
  arithmetic: '算术运算', if_else: '条件分支',

  // ── Strategy / Backtest ──
  strategy: '策略配置', backtest: '回测',

  // ── AI / ML ──
  agent: 'AI 代理',
  feature_engineer: '特征工程', train_model: '训练模型', predict: '模型预测',
  evaluate_model: '模型评估', alpha_mining: 'Alpha 挖掘',
  rl_env: 'RL 环境', rl_train: 'RL 训练', rl_predict: 'RL 预测',

  // ── Trading ──
  place_order: '下单', cancel_order: '取消订单',
  position_query: '持仓查询', order_query: '订单查询',

  // ── Portfolio / Risk ──
  portfolio_summary: '组合概况', allocation: '资产配置',
  risk_metrics: '风险指标', stop_loss: '止损', position_sizer: '仓位计算',
  risk_model: '风险模型', rebalance: '再平衡',

  // ── Notify / Schedule ──
  notify: '通知', alert: '告警',
  schedule: '定时触发', wait: '等待', webhook_trigger: 'Webhook 触发',

  // ── Utility ──
  http_request: 'HTTP 请求', math_op: '数学运算', json_parse: 'JSON 解析',

  // ── Control ──
  loop: '循环', if_condition: '条件判断', sub_workflow: '子工作流',

  // ── Output ──
  log_output: '日志输出', chart_data: '图表输出',

  // ── Research ──
  sentiment: '情绪分析', news_fetcher: '新闻获取', stock_research: '股票研究',
  financials: '财务报表', peer_compare: '同业对比',
  analyst_estimates: '分析师预测', insider_trades: '内部交易',

  // ── Alternative Data ──
  prediction_market: '预测市场', geopolitics: '地缘政治',
  gov_data: '宏观数据', satellite: '卫星数据',
}

export function nodeLabel(type: string): string {
  return NODE_LABELS[type] || type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
