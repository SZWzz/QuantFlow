# i18n Migration: All Panels

## Motivation

50 个面板中仅 SettingsPanel 使用了 vue-i18n，其余全部硬编码字符串。大部分是中文但夹杂英文术语，少量面板全英文（MonteCarlo、ModelRegistry 等）。i18n 基础设施已就绪，需补齐 key 并接入。

## Design

### Phase 1: Expand zh.ts

按 domain 组织，新增 ~400 个 key。Domain 划分：

| Domain | 覆盖面板 |
|--------|---------|
| common | 加载中/暂无数据/保存/取消/刷新/搜索/导出/筛选... |
| kline | K线/分时/昨收/价格/成交量/均价/1m/5m/1d/1w... |
| quote | 开盘/最高/最低/昨收/成交量/市值/市盈率/每股收益/股息率... |
| trade | 买入/卖出/市价/限价/止损/数量/价格/已成交/未成交... |
| portfolio | 总资产/仓位/持仓/盈亏/市值/净值曲线... |
| risk | VaR/CVaR/最大回撤/夏普比率/索提诺比率/年化波动率... |
| schedule | 定时任务/新建任务/Cron/Workflow ID/超时/保存... |
| notify | 通知中心/全部/交易/告警/错误/信息/全部已读... |
| settings | 外观/主题/语言/数据源/API Key/快捷键... |
| broker | 券商连接/已连接/未连接/刷新/市场/模拟交易... |
| news | ─ |
| research | 个股研究/分析师/情绪分析/财务数据/内部交易/同业对比... |
| ml | Alpha 挖掘/因子分析/模型注册/强化学习... |
| geo | 地缘政治风险/风险等级/关联资产/情绪分数... |
| prediction | 预测市场/Yes 概率/交易量/到期/24h 变化... |
| macro | 宏观指标/太阳辐射/能源指标/卫星... |
| workflow | 操作中心/执行日志/画图/回测... |
| monitor | Go 运行时/协程数/堆内存/系统内存/工作流引擎... |
| watchlist | 自选股/代码/最新价/涨跌幅/滚动报价... |
| candlestick | (same as kline, merged) |

### Phase 2: Replace Hardcoded Strings

每个 panel 中所有用户可见文字替换为 `{{ $t('domain.key') }}`。ECharts 中的 tooltip/formatter 文字通过 computed 传参处理。

### Phase 3: en.ts English Translations

为每个 key 提供英文翻译。

## Files

- `frontend/src/lib/i18n/zh.ts` — 扩充至 ~400 key
- `frontend/src/lib/i18n/en.ts` — 对应英文
- `frontend/src/terminal/panels/*.vue` — 50 个面板迁移
- `frontend/src/terminal/*.vue` — SymbolBar/StatusBar/CommandBar 等

## Acceptance Criteria

- [ ] zh.ts 覆盖所有面板 UI 字符串
- [ ] en.ts 有对应英文翻译
- [ ] 所有面板使用 `$t()` 替换硬编码
- [ ] 语言切换后 UI 正确切换
- [ ] `npm run build` 通过
