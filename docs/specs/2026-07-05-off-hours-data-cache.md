# Off-Hours Data Cache — 周末/闭市数据显示上个交易日的数据

## Motivation

周末和闭市时，所有依赖实时数据的面板（行业排名、资金流、异常股票、深度行情、龙虎榜等）返回空/错误状态，不显示上个交易日的数据。用户期待在非交易时间能看到最后一个交易日的完整数据。

已有 `lastQuote`（报价缓存）和 `last_minute_ticks`（分时缓存），需将同类模式扩展到高频面板使用的其他数据类型。

## Data Flow

```
Trading hours:
  Frontend panel → Wails IPC → Go method → adapter fetch → success
    → update OffHoursCache (sync.Map) → saveLastXXX() → disk JSON

Off-hours (weekend / after close):
  Frontend panel → Wails IPC → Go method → check IsTradingHours
    → false → return OffHoursCache data  (or empty)
```

### 文件变更

| File | Change |
|------|--------|
| `internal/market/registry.go` | `IndustryRank` 的 `FetchIndustryRanksWithFallback` 加 `IsTradingHours` 守卫 + `lastRanks` 持久化 |
| `internal/market/offhours.go` | **新文件** — 通用 OffHoursCache 工具（sync.Map + JSON 持久化） |
| `app_market.go` | `GetDepth`, `GetFundFlow` 加 `IsTradingHours` 守卫 + 缓存落盘 |
| `app.go` | `GetIndustryRanks`, `GetDragonTiger`, `GetAbnormalStocks` 加 `IsTradingHours` 守卫 + 缓存落盘 |
| `app.go` (Init) | 注册各缓存的 Save/Load 路径 |

### API 变更

各 Go 方法签名不变。当 `IsTradingHours` 为 false 且缓存命中时，返回缓存数据；缓存未命中时返回 `(nil, "offhours_cache")` 而非 `"market is closed"` 错误。

## 实施范围 (MVP)

仅缓存 **影响面板空白最严重** 的 5 个数据类型：

| 数据类型 | 后端方法 | 面板 | 缓存结构 |
|----------|---------|------|---------|
| 行业排名 | `GetIndustryRanks` | IndustryRankPanel | `map[string][]IndustryRank`（keyed by market） |
| 资金流 | `GetFundFlow` | FundFlowPanel | `{daily: []DailyFlowItem, minute: []MinuteFlowItem}` |
| 深度行情 | `GetDepth` | DepthPanel | `map[string]*DepthSnapshot`（keyed by symbol） |
| 异常股票 | `GetAbnormalStocks` | AbnormalStocksPanel | `map[string][]AbnormalStock`（keyed by market） |
| 龙虎榜 | `GetDragonTiger` | DragonTigerPanel | `map[string][]DragonTigerRecord` |
| 涨跌停 | `GetLimitUpDown` | LimitUpDownPanel | `map[string][]LimitUpDownStock` |

### 通用缓存工具 `offhours.go`

```go
// OffHoursCache provides sync.Map + JSON persistence for off-hours data.
type OffHoursCache[T any] struct {
    mu     sync.Mutex
    data   map[string]T
    path   string
    name   string // for logging
}

func NewOffHoursCache[T any](name string) *OffHoursCache[T]
func (c *OffHoursCache[T]) SetPath(path string)
func (c *OffHoursCache[T]) Load() error       // read from disk
func (c *OffHoursCache[T]) Save() error       // atomic write to disk
func (c *OffHoursCache[T]) Get(key string) (T, bool)
func (c *OffHoursCache[T]) Set(key string, val T)
func (c *OffHoursCache[T]) GetAll() map[string]T
func (c *OffHoursCache[T]) SetAll(data map[string]T)
```

## Acceptance Criteria

- [ ] 周末打开 App，行业排名面板显示周五收盘数据
- [ ] 周末打开 App，资金流面板显示周五收盘数据
- [ ] 周末打开 App，深度行情面板显示周五收盘数据
- [ ] 周末打开 App，异常股票面板显示周五收盘数据
- [ ] 周末打开 App，龙虎榜面板显示周五收盘数据
- [ ] 周末打开 App，涨跌停面板显示周五收盘数据
- [ ] 交易时间正常请求不受影响（实时数据优先）
- [ ] 各数据类型缓存独立存储为 JSON 文件
- [ ] 存量数据升级：首次启动时检查是否有旧 `last_quote.json` 路径
- [ ] 缓存文件放在 `data/offhours/` 子目录，避免与 DB 文件冲突

## Risks / Trade-offs

- 缓存数据量：行业排名约 100 条 × 4 市场，资金流约 5000 条 × 日级，深度每个合约 ~20 档。总 JSON 体积 < 2MB，可接受。
- 缓存时效：`lastQuote` 类似的"最后一次成功"模式。节假日不更新，显示的仍是节前数据，但优于空白。
- 首次运行：若从未在交易时间运行（全新安装），缓存为空，周末仍为空。可接受。
- 反序列化兼容性：`sync.Map` → JSON → `map[string]T` 要求 T 满足 `json.Marshaler/Unmarshaler`，Go 泛型的类型约束需注意。各 Struct 已自带 json tag。
