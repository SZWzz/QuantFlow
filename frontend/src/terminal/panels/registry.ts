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

register('crypto-overview', () => import('./CryptoOverviewPanel.vue'), { label: '加密货币概览', category: '市场行情', description: '主流加密货币行情' })
register('abnormal-stocks', () => import('./AbnormalStocksPanel.vue'), { label: '异动监控', category: '市场行情', description: '全市场异动股票实时监控' })
register('dragon-tiger', () => import('./DragonTigerPanel.vue'), { label: '龙虎榜', category: '市场行情', description: '龙虎榜日榜单与个股上榜记录' })
register('limit-up-down', () => import('./LimitUpDownPanel.vue'), { label: '涨跌停监控', category: '市场行情', description: 'A 股涨停/跌停实时监控' })
register('hk-connect', () => import('./HKConnectPanel.vue'), { label: '港股通', category: '市场行情', description: '北向资金实时流向与额度概览' })
register('funding-rate', () => import('./FundingRatePanel.vue'), { label: '资金费率', category: '市场行情', description: '加密货币永续合约资金费率' })
register('liquidation', () => import('./LiquidationPanel.vue'), { label: '爆仓追踪', category: '市场行情', description: '加密货币爆仓实时追踪' })
register('short-interest', () => import('./ShortInterestPanel.vue'), { label: '做空数据', category: '市场行情', description: '美股做空比例与天数' })

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
register('trading-journal', () => import('./TradingJournalPanel.vue'), { label: '交易日志', category: '组合与风控', description: '逐日 P&L 归因与交易分析' })
register('scenario-analysis', () => import('./ScenarioAnalysisPanel.vue'), { label: '情景分析', category: '组合与风控', description: '组合压力测试与情景模拟' })
register('ipo-calendar', () => import('./IPOCalendarPanel.vue'), { label: '新股日历', category: '市场行情', description: 'A股新股发行申购上市日历' })
register('ex-dividend', () => import('./ExDividendPanel.vue'), { label: '分红除权', category: '市场行情', description: 'A股除权除息日历与股息率' })
register('cb-arbitrage', () => import('./CBArbitragePanel.vue'), { label: '可转债套利', category: '市场行情', description: '可转债溢价率套利与强赎预警' })
register('hk-ipo', () => import('./HKIPOPanel.vue'), { label: '香港IPO', category: '港股', description: '港股新股认购与上市表现' })
register('hk-derivatives', () => import('./HKDerivativesPanel.vue'), { label: '牛熊证/涡轮', category: '港股', description: '香港牛熊证与认股证行情' })
register('hk-settlement', () => import('./HKSettlementPanel.vue'), { label: '港股交收', category: '港股', description: '港股交收规则与费用计算' })
register('us-option-chain', () => import('./USOptionsPanel.vue'), { label: '期权链', category: '美股', description: '美股期权链看涨/看跌矩阵' })
register('wash-sale', () => import('./WashSalePanel.vue'), { label: 'Wash Sale', category: '美股', description: 'IRS 1091 洗售亏损检测' })
register('institutional-trades', () => import('./DarkPoolPanel.vue'), { label: '机构交易', category: '美股', description: 'SEC 文件中的机构与内部人交易' })
register('depth-comparison', () => import('./DepthComparisonPanel.vue'), { label: '多交易所深度对比', category: '加密货币', description: '跨交易所买卖盘深度对比' })
register('defi-tvl', () => import('./DefiTVLPanel.vue'), { label: 'DeFi TVL 排行', category: '加密货币', description: 'DeFi 协议锁仓量排行榜' })
register('whale-tracking', () => import('./WhaleTrackingPanel.vue'), { label: '巨鲸追踪', category: '加密货币', description: '大额链上转账监控' })
register('gas-tracker', () => import('./GasFeePanel.vue'), { label: 'Gas 追踪', category: '加密货币', description: '以太坊实时 Gas 价格' })

// ── 研究分析 ──
register('stock-research', () => import('./StockResearchPanel.vue'), { label: '股票研究', category: '研究分析', description: '多维度个股研究' })
register('financials', () => import('./FinancialsPanel.vue'), { label: '财务报表', category: '研究分析', description: '三年对比+健康评分+异常检测' })
register('valuation', () => import('./ValuationPanel.vue'), { label: 'DCF 估值', category: '研究分析', description: '三情景 DCF 估值+买卖建议' })
register('audit', () => import('./AuditPanel.vue'), { label: '财务审计', category: '研究分析', description: '收入质量/商誉/现金流审计风险检测' })
register('forecast', () => import('./ForecastPanel.vue'), { label: '财务预测', category: '研究分析', description: '线性回归两年三情景预测' })
register('peer-comparison', () => import('./PeerComparisonPanel.vue'), { label: '同业对比', category: '研究分析', description: '同行业公司对比' })
register('analyst-estimates', () => import('./AnalystEstimatesPanel.vue'), { label: '分析师预测', category: '研究分析', description: '分析师一致预期' })
register('insider-trading', () => import('./InsiderTradingPanel.vue'), { label: '内部交易', category: '研究分析', description: '高管/大股东交易' })
register('congress-trading', () => import('./CongressTradingPanel.vue'), { label: '国会议员交易', category: '研究分析', description: '美国国会议员股票交易' })
register('sentiment', () => import('./SentimentPanel.vue'), { label: '市场情绪', category: '研究分析', description: '新闻/社交媒体情绪' })
register('options', () => import('./OptionsPanel.vue'), { label: '期权', category: '研究分析', description: 'BSM 期权定价与希腊值分析' })
register('fundflow', () => import('./FundFlowPanel.vue'), { label: '资金流向', category: '研究分析', description: '龙虎榜、大宗交易、主力资金' })
register('margin', () => import('./MarginPanel.vue'), { label: '融资融券', category: '研究分析', description: '两融余额与沪深港通' })
register('funds', () => import('./FundsPanel.vue'), { label: '基金ETF', category: '研究分析', description: '公募基金、ETF 排名' })
register('futures', () => import('./FuturesPanel.vue'), { label: '期货', category: '研究分析', description: '商品期货与股指期货' })
register('macro', () => import('./GovDataPanel.vue'), { label: '宏观经济', category: '研究分析', description: '中国/美国/全球宏观经济指标（含看涨看跌信号）' })
register('index', () => import('./IndexPanel.vue'), { label: '指数', category: '研究分析', description: '指数行情与成分股' })
register('bonds', () => import('./BondsPanel.vue'), { label: '债券', category: '研究分析', description: '国债、企业债、可转债' })
register('sector-rotation', () => import('./SectorRotationPanel.vue'), { label: '板块轮动', category: '研究分析', description: 'RRG 板块相对强度轮动分析' })
register('economic-calendar', () => import('./EconomicCalendarPanel.vue'), { label: '经济日历', category: '研究分析', description: '全球宏观经济数据发布日历' })
register('earnings-calendar', () => import('./EarningsCalendarPanel.vue'), { label: '财报日历', category: '研究分析', description: '美股财报发布时间与超预期数据' })
register('cross-asset-corr', () => import('./CrossAssetCorrelationPanel.vue'), { label: '跨资产相关性', category: '研究分析', description: '多资产类别相关系数矩阵' })

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
register('satellite', () => import('./SatellitePanel.vue'), { label: '卫星数据', category: '另类数据', description: 'NASA 能源/火灾监测' })

// ── 美股 ──
register('sec_financials', () => import('./SECFinancialsPanel.vue'), { label: 'SEC财报', category: '美股', description: '美股 XBRL 财务报表' })
register('sec_13f', () => import('./SEC13FPanel.vue'), { label: '13F持仓', category: '美股', description: '机构 13F 持仓报告' })

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
