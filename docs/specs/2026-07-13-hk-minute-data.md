# 港股分时数据双通道方案

## Motivation

当前港股分时数据只有 Yahoo 一条通道，Yahoo 从中国请求时频繁返回 HTML（geo-block），导致港股分时线完全不可用。需要增加中国可访问的备用数据源。

## Design

### 整体架构

```
HK 分钟链: ["akshare_hk_minute", "qos", "yahoo"]
                │                │           │
                │                │           └── Yahoo (原通道, geo-block 时自动跳过)
                │                │
                │                └── QOS API (付费, 需配置 KEY)
                │
                └── AKShare stock_hk_hist_min_em (免费, 东财数据源)
                     │
                     └── Python gRPC sidecar
                          └── ak.stock_hk_hist_min_em(symbol="00700", period="1")
```

### A: AKShare Python 侧车方案

**Python 侧** (`python/src/data/fetcher.py`):

新增 `_fetch_akshare_hk_minute(symbol: str) → list[dict]`:
- 调用 `ak.stock_hk_hist_min_em(symbol=symbol, period="1", start_date=today, end_date=today)`
- 将 DataFrame 转换为 `[{time, price, volume}]` 格式
- 注册到 gRPC `data_type = "hk_minute"`

**Go 侧** (`internal/market/adapters/akshare_minuteline.go`):

新增 `AKShareMinuteAdapter`：
- 实现 `MinuteLineProvider` 接口
- 通过 Python DataClient gRPC 调用 `FetchData("akshare", "hk_minute", [symbol])`
- 解析返回的 JSON 为 `[]market.MinuteTick`
- 3 秒 cooldown（与 mootdx 一致）

### B: QOS API 方案

**Go 侧** (`internal/market/adapters/qos_minuteline.go`):

新增 `QOSMinuteAdapter`：
- 实现 `MinuteLineProvider` 接口
- 调用 `POST https://api.qos.hk/kline`
- 请求体: `{"kline_reqs":[{"c":"HK:700","co":240,"a":0,"kt":1}]}`
- 认证: 请求头/参数携带 `key`
- 解析响应中的 `k` 数组为 `[]market.MinuteTick`
- 注册到 `MinuteChains["HK"]["stock"]`

**配置** (`frontend/src/stores/settings.ts` + `internal/config/config.go`):

QOS API Key 通过设置面板配置，存储在 SQLite 配置表：
- `config` 表新增 `qos_api_key` 记录
- 前端 SettingsPanel 新增 `QOS API Key` 输入框
- `GetAPIKey("qos")` 返回存储的值

### 新增/修改文件

| 文件 | 操作 |
|------|------|
| `python/src/data/fetcher.py` | 新增 `_fetch_akshare_hk_minute` |
| `internal/market/adapters/akshare_minuteline.go` | 新建，AKShare 分时适配器 |
| `internal/market/adapters/qos_minuteline.go` | 新建，QOS 分时适配器 |
| `internal/market/adapters/adapter.go` | 注册 `QOSAdapter` 接口断言 |
| `internal/market/registry.go` | `MinuteChains["HK"]` 改为 `["akshare_hk_minute", "qos", "yahoo"]` |
| `app_market.go` | `registerMarketAdapters` 注册新适配器 |
| `internal/config/config.go` | `GetAPIKey("qos")` 支持 |
| `frontend/src/stores/settings.ts` | `qos_api_key` 配置项 |

### 数据流

```
前端请求分时 → GetMinuteLine("00700")
  → FetchMinuteWithFallback("HK", "00700")
    → akshare_hk_minute.FetchMinuteLine("00700")
      → Python gRPC: FetchData("akshare", "hk_minute", ["00700"])
        → ak.stock_hk_hist_min_em(symbol="00700", period="1", ...)
        → 返回 [{time, price, volume}, ...]
      → 解析为 []MinuteTick
    → 失败则重试 qos.FetchMinuteLine("00700")
      → POST https://api.qos.hk/kline {key, kline_reqs: [{c: "HK:700", ...}]}
      → 解析响应
    → 失败则重试 yahoo.FetchMinuteLine("00700")
```

## Acceptance Criteria

- [ ] AKShare Python 侧 `_fetch_akshare_hk_minute` 返回正确格式的 HK 分时数据
- [ ] `akshare_minuteline.go` 通过 gRPC 调用成功并解析为 `[]MinuteTick`
- [ ] `qos_minuteline.go` 正确调用 QOS API 并解析响应
- [ ] QOS API Key 可在前端 SettingsPanel 配置并持久化
- [ ] 两个适配器都注册到 HK 分钟链，Yahoo 作为最后回退
- [ ] 港股分时线在前端 CandlestickPanel 正常显示

## Risks / Trade-offs

- AKShare `stock_hk_hist_min_em` 依赖 Python sidecar 在线；sidecar 不可用时自动跳到 QOS/Yahoo
- QOS 免费套餐每分钟仅 5 次请求，生产使用需付费套餐
- 两个适配器都有 3s cooldown，避免高频请求被限流
- 东财数据有数十秒到数分钟延迟，不适合高频交易决策
