# Watchlist Panel Enhancement

## Motivation
自选股面板当前仅显示代码/名称/价格/涨跌幅四个字段，缺少专业终端必备的数据密度、交互丰富度和状态反馈，落后于同花顺等竞品。

## Design

### P1-1: 扩展显示列

**现状**: 每行只用 5 个 QuoteSnapshot 字段 (symbol/name/last/change/changePct)。
**目标**: 新增 涨速(speed)、量比(volume_ratio)、换手率(turnover_rate)、振幅(amplitude)、成交量(volume)、成交额(amount/10000) 可配置列。

**数据流**: `GetQuote` 返回的 `QuoteSnapshot` 已包含这些字段 → 前端直接取用。

**UI**: 从纯 flex 列表改为 CSS Grid 表格布局，每列可点击排序(asc/desc/none)。用户可通过 header 齿轮按钮选择可见列。

**列定义**:
| 列名 | i18n key | 数据源 | 格式 |
|------|----------|--------|------|
| 代码 | `common.symbol` | `snapshot.symbol` | 文本 |
| 名称 | `common.name` | `snapshot.name` | 文本 |
| 最新价 | `common.price` | `snapshot.last` | `toFixed(2)` |
| 涨跌幅 | `quote.change_pct` | `snapshot.change_pct` | `+2.50%` |
| 涨跌额 | `quote.change` | `snapshot.change` | `+0.50` |
| 涨速 | — | `(last-last_prev)/last_prev` | `+0.12%` |
| 量比 | `kline.volume_ratio` | `snapshot.volume_ratio` | `1.25` |
| 换手率 | `kline.turnover` | `snapshot.turnover_rate` | `2.35%` |
| 振幅 | `kline.amplitude` | `snapshot.amplitude` | `3.45%` |
| 成交量 | `common.volume` | `snapshot.volume` | `12.34万` |
| 成交额 | `quote.turnover` | `snapshot.turnover` | `3.45亿` |
| 最高 | `quote.high` | `snapshot.high` | `1888.00` |
| 最低 | `quote.low` | `snapshot.low` | `1875.00` |

### P1-2: 空状态 / 加载状态

**现状**: 无自选时显示空白滚动区；`watchlist.empty` i18n key 已声明但未使用。
**目标**:
- 空列表: 居中显示 `$t('watchlist.empty')` + 引导文字
- 首次加载: 每行显示 skeleton shimmer 而非 `--`
- 刷新时: 已有数据不闪烁，仅在 header 显示旋转图标

### P2-1: 排序

- 点击列 header 切换排序状态: `none → asc → desc → none`
- 支持降序/升序: 价格、涨跌幅、量比、换手率、振幅、成交量、成交额
- 排序状态通过 header 箭头指示 (`↑`/`↓`)
- 代码和名称不支持排序（保持 localStorage 顺序）

### P2-2: 市场分组

同花顺支持按市场归类。使用 `detectMarket()` 返回值：
- `CN` → A 股
- `HK` → 港股
- `US` → 美股
- `CRYPTO` → 加密

分组为可折叠的 accordion，默认展开。

### P2-3: 交互细节

| 问题 | 修复 |
|------|------|
| 删除按钮不同步 CandlestickPanel | `removeSymbol()` 中 dispatch `watchlist-changed` event |
| 加入自选按钮硬编码中文 | 改为 `$t('watchlist.add')`/`$t('watchlist.remove')` |
| 无右键菜单 | 右键弹出: 删除、跳转K线、复制代码 |
| 无拖拽排序 | 启用 HTML5 Drag & Drop API，拖拽后保存新顺序 |

### P2-4: 实时轮询

**现状**: 无自动更新。
**目标**: 添加 10 秒轮询定时器，仅在组件可见时运行。使用 `document.hidden` API 在 tab 切换时暂停。

### Files Changed

| File | Change |
|------|--------|
| `frontend/src/terminal/panels/WatchlistPanel.vue` | 重写为 CSS Grid 表格 + 空状态 + 排序 + 分组 + 轮询 + 右键菜单 + 拖拽 |
| `frontend/src/terminal/panels/CandlestickPanel.vue` | 加入自选按钮改用 i18n；`removeSymbol` dispatch event |
| `frontend/src/lib/i18n/zh.ts` | 新增 `watchlist.add/remove/sort_by/column/speed` 等 key |
| `frontend/src/lib/i18n/en.ts` | 同上英文 |
| `frontend/src/terminal/panels/__tests__/WatchlistPanel.test.ts` | 扩展测试覆盖 |
| `CHANGELOG.md` | 更新条目 |

### Acceptance Criteria
- [ ] CSS Grid 表格布局，可点击列 header 排序
- [ ] 排序状态指示器 (↑/↓)
- [ ] 列配置弹窗，可开关显示列
- [ ] 空状态显示引导文字
- [ ] 首次加载 skeleton shimmer
- [ ] 市场分组 accordion (CN/HK/US/CRYPTO)
- [ ] 右键菜单 (删除/跳转K线/复制代码)
- [ ] 拖拽排序
- [ ] 10 秒轮询，tab 隐藏时暂停
- [ ] 删除自选时 CandlestickPanel 按钮同步更新
- [ ] 加入自选按钮使用 i18n
- [ ] 构建通过，Python/Go 测试通过

### Risks / Trade-offs
- 排序和分组状态存在 localStorage（`quantflow-watchlist-config`），避免每次打开重置
- 轮询间隔 10 秒平衡实时性和 API 调用频率（`TickerTapePanel` 也是 10 秒）
- CSS Grid 布局可能对大数量自选（>100）产生性能开销，但 real-world 场景极少超过 50
