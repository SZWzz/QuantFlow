# 正规化二期：适配器接入 NormalizeVolume

## Motivation

正规化一期建立了 `internal/normalize/` 包（统一 `OHLCVBar` 类型、`NormalizeVolume()` 函数、`FieldMapper` 接口），但**未修改任何适配器**。当前 6 个 A 股适配器的成交量归一化处理严重不一致：

### 现状矩阵

| 适配器 | OHLCV Volume | Quote Volume | Depth Size | 方式 |
|--------|-------------|-------------|------------|------|
| **eastmoney** | ✅ `* 100` | ❌ 无归一化 | N/A | 手写硬编码 |
| **mootdx** | ✅ `* 100` | ❌ 无归一化 | N/A | 手写硬编码 |
| **tushare** | ✅ `* 100` | ✅ `* 100` | N/A | 手写硬编码 |
| **sina** | ❌ 无归一化 | ❌ 无归一化 | ✅ `* 100` | 深度有，行情/K线无 |
| **tencent** | ❌ 无归一化 | ❌ 无归一化 | ✅ `* 100` | 深度有，行情/K线无 |
| **baidu** | ❌ 无归一化 | ❌ 无归一化 | N/A | 全无 |

### 问题严重性

1. **数据不一致**：同一只股票通过 eastmoney 获取的 K 线成交量是通过 sina 获取的 100 倍
2. **硬编码分散**：6 处手写 `* 100` 散落在 4 个文件中，无统一入口
3. **normalize 包形同虚设**：`volume.go` 已定义全部 6 个源的 ×100 映射，但只有 `OHLCVMapper.Parse()` 内部调用了 `NormalizeVolume()`
4. **部分字段遗漏**：eastmoney/mootdx 的 OHLCV 已归一化但 Quote 未归一化，同一适配器内数据不一致

## Design

### 核心原则

所有 A 股数据源的成交量统一通过 `normalize.NormalizeVolume(adapter.Name(), volume)` 转换，消除手写 `* 100`。

### 修改范围

**涉及文件（9 个修改 + 2 个新建）：**

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/market/adapters/eastmoney.go` | 修改 | OHLCV: `* 100` → `NormalizeVolume`；Quote: 添加 NormalizeVolume |
| `internal/market/adapters/mootdx.go` | 修改 | OHLCV: `* 100` → `NormalizeVolume`；Quote: 添加 NormalizeVolume |
| `internal/market/adapters/tushare.go` | 修改 | OHLCV + Quote: `* 100` → `NormalizeVolume` |
| `internal/market/adapters/sina.go` | 修改 | OHLCV CN/HK + Quote CN/HK/US: 添加 NormalizeVolume；Depth: `* 100` → NormalizeVolume |
| `internal/market/adapters/tencent.go` | 修改 | OHLCV + Quote: 添加 NormalizeVolume；Depth: `* 100` → NormalizeVolume |
| `internal/market/adapters/baidu.go` | 修改 | OHLCV + Quote: 添加 NormalizeVolume |
| `internal/normalize/volume.go` | 修改 | 添加 `VolumeMultiplier()` 公开函数（供测试查询） |
| `internal/market/adapters/volume_normalize_test.go` | 新建 | 适配器成交量归一化集成验证 |
| `internal/normalize/volume_test.go` | 修改 | 扩展测试用例 |
| `internal/market/adapters/ohlcv_test.go` | 修改 | 验证 OHLCV 成交量归一化 |
| `CHANGELOG.md` | 修改 | 记录变更 |

### 变更示例

```go
// Before (eastmoney.go:213)
Volume: parseFloatSafe(fields[5]) * 100,

// After
Volume: normalize.NormalizeVolume(a.Name(), parseFloatSafe(fields[5])),
```

```go
// Before (mootdx.go:150)
Volume: b.Volume * 100, // 手→shares

// After
Volume: normalize.NormalizeVolume(a.Name(), b.Volume),
```

```go
// Before (sina.go:269) — 未归一化
Volume:    volume,

// After — 添加归一化（sina CN 行情返回手）
Volume:    normalize.NormalizeVolume(a.Name(), volume),
```

### 数据流

```
适配器 FetchOHLCV / FetchQuote / FetchDepth
    │
    ├─ 原始 volume 值（手, lots）
    │
    ▼
normalize.NormalizeVolume(adapter.Name(), rawVolume)
    │
    ├─ volumeMultiplier[source] → ×100 (A 股 6 源)
    ├─ 未知源 → 原值返回 (美股/加密)
    │
    ▼
market.OHLCVBar / QuoteSnapshot / DepthLevel (统一 shares)
```

## Acceptance Criteria

- [ ] 所有 6 个 A 股适配器中不再出现手写 `* 100`，全部替换为 `normalize.NormalizeVolume()`
- [ ] eastmoney/mootdx/tushare/sina/tencent/baidu 的 OHLCV Volume 字段通过 NormalizeVolume
- [ ] eastmoney/mootdx/tushare/sina/tencent/baidu 的 Quote Volume 字段通过 NormalizeVolume
- [ ] sina/tencent 的 Depth Size 字段通过 NormalizeVolume（替换现有 `* 100`）
- [ ] 非 A 股适配器（binance/yahoo/finnhub/polygon/alpaca）不受影响
- [ ] `go vet` + `go test ./internal/market/adapters/...` 通过
- [ ] `go test ./internal/normalize/...` 通过
- [ ] 新增测试验证同一股票通过不同适配器获取的成交量值一致（±5% 容差，因为不同源可能有微小差异）

## Risks / Trade-offs

- **Sina/Tencent/Baidu 当前未归一化**：添加 `NormalizeVolume` 会使其返回值扩大 100 倍。需确认这些源确实返回「手」而非「股」。根据 `volume.go` 的映射表和已有的深度数据归一化实践（sina/tencent 深度已做 ×100），这些源返回手是确定的。
- **下游兼容**：所有消费 OHLCV/Quote 的上层代码（面板、回测引擎、策略）已经预期接收 shares 单位的成交量（因为 eastmoney 已经在归一化），sina/tencent/baidu 的修复会消除现有的 100 倍偏差。
- **不涉及非 A 股源**：binance/yahoo/finnhub/polygon/alpaca/gateio/okx/coingecko 等返回的是原生 shares/coins，`NormalizeVolume` 对未知源返回原值，安全无副作用。
- **不修改 workflow nodes**：`DataNormalizeNode` 已经在 `OHLCVMapper.Parse()` 中调用 `NormalizeVolume`，无需修改。
