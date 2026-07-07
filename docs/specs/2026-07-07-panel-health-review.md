# Panel Health Review — Build & Runtime Bug Fixes

## Motivation

Panel 检查发现了 3 类问题：

1. **Go 后端** — 测试 panic (`a.cfg` nil dereference) + 编译错误 (`%w` on non-wrapping error)
2. **Vue TypeScript** — 15 个编译错误分布在 9 个面板，包括 marked API 不兼容、类型窄化错误、缺少 i18n 解构
3. **测试基础设施** — 47/65 测试因 `vue-i18n` 未在测试 wrapper 安装而失败

这些问题阻塞了 `npx vue-tsc --noEmit` 和 `go test ./...` 通过，需要一次性修复。

## Design

### 数据流概览

```
Issues grouped by domain:
├── Go backend (2 files, 2 fixes)
│   ├── app_test.go          — nil cfg on App{}
│   └── eastmoney_signals.go — %w → %v
├── Vue panels (9 files, 11 fixes)
│   ├── AIChatPanel.vue         — marked Renderer.code 签名
│   ├── CandlestickPanel.vue    — d[0] → d.date
│   ├── CorrelationPanel.vue    — corrMatrix 类型
│   ├── DefiTVLPanel.vue        — sort bVal 窄化
│   ├── DistributionPanel.vue   — 加入 useI18n()
│   ├── IndicatorPanel.vue      — align type
│   ├── LimitUpDownPanel.vue    — switchMarket 签名
│   ├── MarketOverviewPanel.vue — switchMarket 签名
│   └── WhaleTrackingPanel.vue  — sort bVal 窄化
├── Test infra (no code change, requires setup.ts)
```

### Go Backend Fixes

#### 1. `app_test.go` — nil cfg in App struct

**Root cause**: `TestApp_RegisterMarketAdapters_AllWired` 构造 `App{marketReg, bridge}` 未设 `cfg`，`registerMarketAdapters()` 调用 `a.cfg.GetAPIKey()` 时 nil dereference。

**Fix**: 在测试里注入 `config.DefaultConfig()`:

```go
a := &App{
    marketReg: market.NewAdapterRegistry(),
    bridge:    nil,
    cfg:       config.DefaultConfig(),
}
```

#### 2. `eastmoney_signals.go` — `%w` on `NewTransientErrorf`

**Root cause**: `market.NewTransientErrorf()` 内部用 `fmt.Sprintf` 构造 `transientError{msg}`，不支持 error-wrapping directive `%w`。

**Fix**: 两处调用将 `%w` 改为 `%v`。transient error 是重试标记而非错误链，不需要 wrap。

```go
// before
market.NewTransientErrorf("eastmoney_signals industry: %w", err)
// after
market.NewTransientErrorf("eastmoney_signals industry: %v", err)
```

### Vue TypeScript Fixes

#### 3. `AIChatPanel.vue` — marked Renderer.code 签名

**Root cause**: marked v18+ 的 `Renderer.code` 从 `(code, language)` 变为 `({ text, lang, escaped })`。

**Fix**: 改用解构参数：

```ts
// before
renderer.code = function (code: string, language: string | undefined) {
// after
renderer.code = function ({ text, lang }: { text: string; lang?: string }) {
```

同时更新内部引用 `code` → `text`, `language` → `lang`。

#### 4. `CandlestickPanel.vue` — `d[0]` should be `d.date`

**Root cause**: `KlineDataItem` 是 `{ date, open, high, low, close, volume }` 对象类型，`jumpToDate()` 用 `d[0]` 访问日期，但对象没有数字索引签名。

**Fix**: `d[0]` → `d.date`，并将 `timestamps` 改为 `string[]`（日期字符串）。

```ts
// before
const timestamps = ohlcvData.value.map(d => d[0])
// after
const timestamps = ohlcvData.value.map(d => d.date)
```

#### 5. `CorrelationPanel.vue` — corrMatrix 空类型

**Root cause**: `fetchWithCache` 返回 `corrMatrix` 被推断为 `{}`（空对象），`corrMatrix?.[si]?.[sj]` 报错。

**Fix**: 在 `fetchWithCache` 的泛型参数中明确指定返回类型：

```ts
const { data: corrMatrix } = await fetchWithCache<Record<string, Record<string, number>>>(key, ...)
```

#### 6. `DefiTVLPanel.vue` & `WhaleTrackingPanel.vue` — sort 中 bVal 未窄化

**Root cause**: `aVal - bVal` 中 `aVal` 被 `typeof === 'number'` 窄化到 `number`，但 `bVal` 仍是 `string | number`，TS 不允许减去 `string`。

**Fix**: 将 `bVal` 也转为 `number`，或在三元中加上类型断言：

```ts
// option A — 窄化 bVal
return (typeof aVal === 'number' ? aVal - (bVal as number) : ...)
// option B — Number() 统一转换 (推荐)
const numA = Number(aVal), numB = Number(bVal)
return (isNaN(numA) || isNaN(numB) ? String(aVal).localeCompare(String(bVal)) : numA - numB) * sortDir.value
```

#### 7. `DistributionPanel.vue` — 缺少 `const { t } = useI18n()`

**Root cause**: `import { useI18n } from 'vue-i18n'` 存在但从未解构 `t`。

**Fix**: 在 `setup` 中添加 `const { t } = useI18n()`（约第 48 行）。

#### 8. `IndicatorPanel.vue` — `align: 'right'` 类型不兼容

**Root cause**: Column 类型定义要求 `align: 'left'`，但代码传了 `'right'`。

**Fix**: 将 `align` 的 `as const` 类型改为兼容的值，或放宽 Column 类型的定义。查看 `PanelTable` 的 Column 类型，如果确实只支持 `'left'`，则:

```ts
// before
align: 'right' as const,
// after: 移除 right 对齐，用 left (PanelTable 的局限)
// 或者: 将 Column 类型扩展为支持 'left' | 'right' | 'center'
```

#### 9. `LimitUpDownPanel.vue` & `MarketOverviewPanel.vue` — switchMarket 签名

**Root cause**: `PanelHeader` emit `tabChange` 类型是 `(key: string)`，但 `switchMarket` 接受窄类型（`'SH'|'SZ'` / `'HK'|'US'|'CN'`）。

**Fix**: 统一改为用 `string` 类型接收后在函数内做运行时校验：

```ts
// before
function switchMarket(mkt: 'SH' | 'SZ') {
// after
function switchMarket(mkt: string) {
  if (mkt !== 'SH' && mkt !== 'SZ') return
```

### Test Infrastructure Fix

#### 10. `vue-i18n` 未在测试 wrapper 安装

**Root cause**: 47 个组件测试 `mount()` 时未先 `app.use(i18n)`，vue-i18n 报 `Need to install with app.use function`。

**Fix**: 在 `frontend/src/__tests__/setup.ts` 或每个测试文件的 `mount()` 前注册 i18n。推荐方案——在 `setup.ts` 集中处理。

由于这涉及测试基础架构且影响范围大，但改动小，在同一个 spec 中完成。

## Acceptance Criteria

- [ ] `go build ./...` 无错误通过
- [ ] `go test ./...` 无 panic 和编译错误（仅预期的 key-not-found 类错误可接受）
- [ ] `npx vue-tsc --noEmit` 无错误通过
- [ ] `npx vitest run` 从 65→至少 100+ passed（i18n 修复后）
- [ ] 9 个面板的 TypeScript 错误全部消除
- [ ] `jumpToDate()` 在 CandlestickPanel 中正确使用日期字符串
- [ ] 4 个使用 switchMarket 的面板正确响应 tab 切换

## Risks / Trade-offs

- **`%w` → `%v`**: 丢失了对原始 error 的 unwrap 能力。但 `NewTransientErrorf` 的语义本就是重试标记而非错误链，不影响上层 `RetryWithBudget`。
- **CorrelationPanel 类型修复**: `Record<string, Record<string, number>>` 是合理近似，但后端可能返回不同结构。如果后端格式变化，需要单独适配。
- **Test i18n**: 所有组件测试需要至少 `app.use(i18n)`，如果测试调用了 `useI18n()` 但未挂载到 app 则会失败。少数测试可能还需要 mock translations。
- **`type: 'right'` 对齐**: 如果 `PanelTable` Column 类型确实只允许 `'left'`，可能需要扩展 Column 定义以支持 `'right' | 'center'`。这涉及 `PanelTable` 的样式逻辑。
