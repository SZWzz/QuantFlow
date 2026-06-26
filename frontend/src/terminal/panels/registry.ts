import { defineAsyncComponent, type Component } from 'vue'

export interface PanelMeta {
  id: string
  label: string
  category: string
  description: string
}

const panelRegistry = new Map<string, Component>()
const panelMeta = new Map<string, PanelMeta>()

function register(id: string, loader: () => Promise<any>, meta: Omit<PanelMeta, 'id'>) {
  panelRegistry.set(id, defineAsyncComponent(loader))
  panelMeta.set(id, { id, ...meta })
}

// ── 市场行情 ──
register('watchlist', () => import('./WatchlistPanel.vue'), { label: '自选股', category: '市场行情', description: '自选股列表 + 实时报价' })
register('quote-detail', () => import('./QuoteDetailPanel.vue'), { label: '行情详情', category: '市场行情', description: '单股详细报价' })
register('candlestick', () => import('./CandlestickPanel.vue'), { label: 'K 线图', category: '市场行情', description: '历史 K 线图' })
register('market-overview', () => import('./MarketOverviewPanel.vue'), { label: '市场概况', category: '市场行情', description: '大盘指数概览' })
register('market-depth', () => import('./MarketDepthPanel.vue'), { label: '市场深度', category: '市场行情', description: '五档买卖盘口' })
register('heatmap', () => import('./HeatmapPanel.vue'), { label: '板块热力图', category: '市场行情', description: '行业板块涨跌热力' })
register('ticker-tape', () => import('./TickerTapePanel.vue'), { label: '滚动报价条', category: '市场行情', description: '滚动实时行情' })
register('crypto-overview', () => import('./CryptoOverviewPanel.vue'), { label: '加密货币概览', category: '市场行情', description: '主流加密货币行情' })
register('abnormal-stocks', () => import('./AbnormalStocksPanel.vue'), { label: '异动监控', category: '市场行情', description: '全市场异动股票实时监控' })

// ── 交易执行 ──
register('order-entry', () => import('./OrderEntryPanel.vue'), { label: '下单面板', category: '交易执行', description: '买入/卖出下单' })
register('order-blotter', () => import('./OrderBlotterPanel.vue'), { label: '订单簿', category: '交易执行', description: '委托订单列表' })
register('execution', () => import('./ExecutionPanel.vue'), { label: '成交明细', category: '交易执行', description: '成交执行详情' })
register('basket-order', () => import('./BasketOrderPanel.vue'), { label: '篮子订单', category: '交易执行', description: '批量下单' })
register('broker-status', () => import('./BrokerStatusPanel.vue'), { label: '券商状态', category: '交易执行', description: '券商连接状态' })
register('action-center', () => import('./ActionCenterPanel.vue'), { label: '操作中心', category: '交易执行', description: '交易通知和操作' })

// ── 组合与风控 ──
register('position', () => import('./PositionPanel.vue'), { label: '持仓明细', category: '组合与风控', description: '当前持仓列表' })
register('position-detail', () => import('./PositionDetail.vue'), { label: '持仓详情', category: '组合与风控', description: '单只持仓详细分析' })
register('portfolio-summary', () => import('./PortfolioSummary.vue'), { label: '组合概况', category: '组合与风控', description: '组合整体表现' })
register('trade-history', () => import('./TradeHistory.vue'), { label: '交易记录', category: '组合与风控', description: '历史交易成交记录' })
register('risk-dashboard', () => import('./RiskDashboard.vue'), { label: '风险仪表盘', category: '组合与风控', description: '组合风险指标一览' })
register('rebalance', () => import('./RebalancePanel.vue'), { label: '再平衡', category: '组合与风控', description: '组合再平衡建议' })
register('broker-config', () => import('./BrokerConfig.vue'), { label: '券商配置', category: '组合与风控', description: '券商账户设置' })

// ── 图表分析 ──
register('equity-curve', () => import('./EquityCurvePanel.vue'), { label: '权益曲线', category: '图表分析', description: '净值曲线' })
register('surface-chart', () => import('./SurfaceChartPanel.vue'), { label: '波动率曲面', category: '图表分析', description: '隐含波动率曲面' })
register('correlation', () => import('./CorrelationPanel.vue'), { label: '相关性矩阵', category: '图表分析', description: '多标的相关性' })
register('distribution', () => import('./DistributionPanel.vue'), { label: '收益分布', category: '图表分析', description: '收益率分布直方图' })
register('drawing', () => import('./DrawingPanel.vue'), { label: '绘图工具', category: '图表分析', description: '自由绘图标注' })
register('monte-carlo', () => import('./MonteCarloPanel.vue'), { label: '蒙特卡洛', category: '图表分析', description: '蒙特卡洛模拟' })

// ── 研究分析 ──
register('stock-research', () => import('./StockResearchPanel.vue'), { label: '股票研究', category: '研究分析', description: '多维度个股研究' })
register('financials', () => import('./FinancialsPanel.vue'), { label: '财务报表', category: '研究分析', description: '三大报表数据' })
register('peer-comparison', () => import('./PeerComparisonPanel.vue'), { label: '同业对比', category: '研究分析', description: '同行业公司对比' })
register('analyst-estimates', () => import('./AnalystEstimatesPanel.vue'), { label: '分析师预测', category: '研究分析', description: '分析师一致预期' })
register('insider-trading', () => import('./InsiderTradingPanel.vue'), { label: '内部交易', category: '研究分析', description: '高管/大股东交易' })
register('congress-trading', () => import('./CongressTradingPanel.vue'), { label: '国会议员交易', category: '研究分析', description: '美国国会议员股票交易' })
register('sentiment', () => import('./SentimentPanel.vue'), { label: '市场情绪', category: '研究分析', description: '新闻/社交媒体情绪' })

// ── 量化分析 ──
register('backtest-result', () => import('./BacktestResultPanel.vue'), { label: '回测结果', category: '量化分析', description: '策略回测绩效' })
register('factor-analysis', () => import('./FactorAnalysisPanel.vue'), { label: '因子分析', category: '量化分析', description: '多因子分析' })
register('model-registry', () => import('./ModelRegistryPanel.vue'), { label: '模型注册', category: '量化分析', description: 'ML 模型管理' })
register('prediction-dashboard', () => import('./PredictionDashboardPanel.vue'), { label: '预测面板', category: '量化分析', description: '模型预测结果' })
register('alpha-mining', () => import('./AlphaMiningWorkspacePanel.vue'), { label: 'Alpha 挖掘', category: '量化分析', description: 'Alpha 因子挖掘' })
register('rl-monitor', () => import('./RLMonitorPanel.vue'), { label: 'RL 监控', category: '量化分析', description: '强化学习训练监控' })

// ── 另类数据 ──
register('prediction-market', () => import('./PredictionMarketPanel.vue'), { label: '预测市场', category: '另类数据', description: 'Polymarket 预测市场' })
register('geopolitics', () => import('./GeopoliticsPanel.vue'), { label: '地缘政治', category: '另类数据', description: 'GDELT 地缘风险' })
register('gov-data', () => import('./GovDataPanel.vue'), { label: '宏观数据', category: '另类数据', description: 'FRED 经济指标' })
register('satellite', () => import('./SatellitePanel.vue'), { label: '卫星数据', category: '另类数据', description: 'NASA 能源/火灾监测' })

// ── 系统 ──
register('ai-chat', () => import('./AIChatPanel.vue'), { label: 'AI 对话', category: '系统', description: 'AI 助手对话' })
register('news', () => import('./NewsPanel.vue'), { label: '新闻流', category: '系统', description: '实时财经新闻' })
register('system-monitor', () => import('./SystemMonitorPanel.vue'), { label: '系统监控', category: '系统', description: 'CPU/内存/网络' })
register('schedule-panel', () => import('./SchedulePanel.vue'), { label: '定时任务', category: '系统', description: '自动化定时任务' })
register('notify-panel', () => import('./NotifyPanel.vue'), { label: '通知中心', category: '系统', description: '消息通知列表' })
register('settings', () => import('./SettingsPanel.vue'), { label: '系统设置', category: '系统', description: '全局配置' })

// ── 欢迎页 ──
register('welcome', () => import('./WelcomePanel.vue'), { label: '欢迎', category: '系统', description: '欢迎页面' })

// ── 量化分析 (继续) ──
register('chanlun', () => import('./ChanlunPanel.vue'), { label: '缠论分析', category: '量化分析', description: '缠中说禅技术分析' })
register('indicator', () => import('./IndicatorPanel.vue'), { label: '技术指标', category: '量化分析', description: '19项技术指标计算' })
register('stock-scanner', () => import('./StockScannerPanel.vue'), { label: '选股扫描', category: '量化分析', description: '多策略全市场选股' })

export function getPanelComponent(panelId: string): Component | undefined {
  return panelRegistry.get(panelId)
}

export function getRegisteredPanels(): string[] {
  return Array.from(panelRegistry.keys())
}

export function getPanelMeta(panelId: string): PanelMeta | undefined {
  return panelMeta.get(panelId)
}

export function getAllPanelMeta(): PanelMeta[] {
  return Array.from(panelMeta.values())
}

export function getPanelsByCategory(): Record<string, PanelMeta[]> {
  const groups: Record<string, PanelMeta[]> = {}
  for (const meta of panelMeta.values()) {
    if (!groups[meta.category]) groups[meta.category] = []
    groups[meta.category].push(meta)
  }
  return groups
}
