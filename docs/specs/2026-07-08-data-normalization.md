# 数据归一化系统 (Data Normalization)

## Motivation

当前 QuantFlow 内部存在三种数据重复定义、各适配器各自为政处理单位转换、无统一字段映射层的架构债务：

1. **重复类型**：`market.OHLCVBar` / `trading.OHLCVBar` / `data.ohlcvParquetRow` 结构相同但分属三包
2. **分散的单位转换**：6 个适配器（EastMoney/Sina/Tencent/TuShare/Mootdx/Baidu）各自做 `volume * 100`（手→股），无集中的 `NormalizeVolume()`
3. **无报价字段映射层**：每个适配器从原始 CSV/JSON 位置索引硬编码映射到 `QuoteSnapshot`
4. **无订单类型/状态映射注册表**：每个券商适配器手写自己的 `type↔enum` 映射函数

## Design

### 核心模式：`internal/normalize/` 包

新建 `internal/normalize/` 包，提供：

- **统一 Schema 类型**：`OHLCVBar`（单一真相源），其他包 import 此类型
- **FieldMapper 接口 + 实现**：定义 `Mapper[T any]` 接口，为每个源实现映射器
- **Normalizer 工具函数**：成交量归一化、价格精度归一化、时间戳格式化

### Schema 定义（第一阶段 4 个，覆盖 80% 场景）

| Schema | 类型 | 说明 |
|--------|------|------|
| OHLCV | `normalize.OHLCVBar` | 替代 3 个重复定义 |
| Quote | `normalize.Quote` | 标准化快照字段 |
| OrderStatus | `normalize.OrderStatusMapper` | 订单状态映射注册表 |
| OrderType | `normalize.OrderTypeMapper` | 订单类型映射注册表 |

### 数据流

```
原始源数据 (CSV/JSON/Protobuf)
    ↓
FieldMapper.Parse(raw) → normalized type
    ↓
消费方使用 normalized type（不再关心源格式）
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/normalize/ohlcv.go` | 新建 | 统一 OHLCVBar 类型 + NormalizeOHLCV |
| `internal/normalize/ohlcv_test.go` | 新建 | 测试 |
| `internal/normalize/volume.go` | 新建 | NormalizeVolume(source, vol, price) |
| `internal/normalize/volume_test.go` | 新建 | 测试 |
| `internal/normalize/mapper.go` | 新建 | FieldMapper 接口 + OrderStatus/Type 注册表 |
| `internal/normalize/mapper_test.go` | 新建 | 测试 |
| `internal/market/types.go` | 修改 | OHLCVBar 改为 alias `normalize.OHLCVBar` |
| `internal/trading/types.go` | 修改 | OHLCVBar 改为 alias `normalize.OHLCVBar` |
| `internal/workflow/nodes/register.go` | 修改 | 注册 DataNormalizeNode |
| `internal/workflow/nodes/data_normalize.go` | 新建 | DataNormalizeNode 实现 |
| `internal/workflow/nodes/data_normalize_test.go` | 新建 | 测试 |

### API 变更

```go
package normalize

// OHLCVBar is the single source of truth for OHLCV data.
type OHLCVBar struct {
    Symbol string
    Date   string  // "2006-01-02"
    Open   float64
    High   float64
    Low    float64
    Close  float64
    Volume float64
}

// NormalizeVolume converts trading volume to standard shares.
// source is adapter name (e.g. "eastmoney", "sina").
func NormalizeVolume(source string, volume float64) float64

// Mapper defines a generic field mapper from raw data to a normalized type.
type Mapper[T any] interface {
    // Parse converts raw source data into the normalized type.
    Parse(raw map[string]any) (T, error)
    // Source returns the adapter/source name.
    Source() string
}

// OrderStatusMapper maps broker-specific status strings to trading.OrderStatus.
type OrderStatusMapper struct{ broker string }

func (m *OrderStatusMapper) Map(status string) trading.OrderStatus
```

## Acceptance Criteria

- [ ] `normalize.OHLCVBar` 定义，`market`/`trading` 包改为 alias
- [ ] `normalize.NormalizeVolume()` 覆盖 6 个 A 股适配器
- [ ] `normalize.OrderStatusMapper` + `OrderTypeMapper` 覆盖 IBKR/Binance/Alpaca
- [ ] `normalize.Mapper[T]` 接口定义 + OHLCV 实现
- [ ] DataNormalizeNode 工作流节点
- [ ] `go vet` + `go test` 通过
- [ ] 向后兼容：消费方代码无需修改

## Risks / Trade-offs

- **不修改适配器**：第一阶段只建立类型和工具，不改动现有适配器实现
- **不涉及报价字段**：QuoteSnapshot 字段映射（25 字段）涉及面太大，留作二期
- **只覆盖 A 股成交量**：加密/美股的成交量已经是 shares，无需转换
