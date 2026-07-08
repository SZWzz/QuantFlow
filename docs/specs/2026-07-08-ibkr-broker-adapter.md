# IBKR Broker Adapter — Interactive Brokers 集成

## Motivation

QuantFlow 当前只支持 Alpaca（美股）和 Binance（加密）两家实盘券商，缺少覆盖港股/美股/期权/期货全市场的关键 broker。IBKR (Interactive Brokers) 是港美股量化交易的事实标准，支持港股、美股、欧股、期货、期权、外汇等多资产类别，是 QuantFlow 走向实盘交易的核心缺口。

## Design

### Architecture

新增 IBKR broker adapter，实现 `internal/trading/broker.go` 定义的 `Broker` 接口。通过 **IBKR Client Portal Web API**（REST on localhost）与 IB Gateway 通信。

```
QuantFlow App
    │
    ├── internal/trading/broker.go        ← Broker 接口（不变）
    ├── internal/trading/brokers/ibkr.go  ← IBKR 实现（新增）
    ├── internal/trading/brokers/ibkr_test.go
    │
    │   HTTPS (自签名证书, InsecureSkipVerify)
    │   ─────────→ localhost:5000/v1/api/
    │              (IB Gateway / Client Portal)
    │
    └── internal/auth/credential.go       ← 复用加密存储存配置
```

### New / Modified Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/trading/brokers/ibkr.go` | CREATE | IBKR broker implementation |
| `internal/trading/brokers/ibkr_test.go` | CREATE | Unit tests with HTTP mocks |
| `internal/trading/brokers/ibkr_config.go` | CREATE | Config struct + env/credential loading |
| `internal/trading/brokers/ibkr_session.go` | CREATE | Session management + refresh goroutine |
| `app_startup.go` | MODIFY | Register IBKR broker if configured |
| `internal/trading/brokers/ibkr_types.go` | CREATE | IBKR-specific API response types |

### Broker 实现

```go
type IBKRConfig struct {
    Host      string `json:"host"`       // default: localhost
    Port      int    `json:"port"`       // default: 5000
    AccountID string `json:"account_id"` // IBKR numeric account ID
    CredKey   string `json:"-"`          // key in credentials table
}

type IBKRBroker struct {
    cfg       IBKRConfig
    client    *http.Client
    session   string          // sso session token
    expiresAt time.Time
    connected bool
    mu        sync.RWMutex
    orderCbs  []func(*trading.Order)
    tradeCbs  []func(*trading.Trade)
}
```

### Broker 接口 ↔ IBKR API 映射

| Broker 接口 | IBKR CP API | 说明 |
|---|---|---|
| `Connect` | `GET /sso/validate` + `POST /sso/validate` | 检测/初始化 session |
| `IsConnected` | session 未过期 | 本地判断，不发起网络请求 |
| `Disconnect` | `POST /logout` | 清除 session |
| `SubmitOrder` | `POST /iserver/account/{id}/orders` | 提交订单 |
| `CancelOrder` | `DELETE /iserver/account/{id}/order/{id}` | 撤单 |
| `ModifyOrder` | `POST /iserver/account/{id}/order/{id}` | 修改订单 |
| `GetOrders` | `GET /iserver/account/{id}/orders` | 获取订单列表 |
| `GetPositions` | `GET /portfolio/{id}/positions/0` | 获取持仓 |
| `GetAccount` | `GET /portfolio/{id}/summary` | 获取账户信息 |
| `OnOrderUpdate` | N/A (REST 无推送) | 通过轮询 `GetOrders` 模拟 |
| `OnTradeUpdate` | N/A (REST 无推送) | 通过轮询 `GetOrders` 模拟 |

### Session 管理

IBKR CP API 使用 session-based auth。流程：

1. `Connect()`: 先 `GET /sso/validate` 检查是否已有 session
2. 如果返回 200 + session 有效，复用
3. 如果返回失败，提示用户在 IB Gateway 登录后重试
4. 启动后台 goroutine，每 4 分钟 `GET /sso/validate` 保活
5. `Disconnect()`: `POST /logout` + 停止保活 goroutine

Token 不持久化——每次应用启动需要 IB Gateway session。

### 订单映射

IBKR 订单类型比当前系统丰富。初始仅映射三种基础类型：

| 系统 OrderType | IBKR `orderType` | 额外字段 |
|---|---|---|
| `market` | `MKT` | 无 |
| `limit` | `LMT` | `limitPrice` |
| `stop` | `STP` | `stopPrice` |

IBKR 回复中的 order status 映射：

| IBKR Status | 系统 Status |
|---|---|
| `Submitted`, `PreSubmitted` | `pending` |
| `Filled` | `filled` |
| `Cancelled` | `cancelled` |
| `Inactive`, `PendingCancel` | `pending` |
| `ApiCancelled` | `cancelled` |

### Error Handling

- HTTP 401/403 → session 过期，标记 disconnected，下次 `Connect()` 会自动重新认证
- HTTP 5xx → 返回 `BrokerOrderResult` 带错误 Message，订单标记 `rejected`
- 网络超时 → 返回错误，订单保持 `pending` 状态

### Testing

使用 `net/http/httptest` mock IBKR CP API 服务器。

| Test | What it tests |
|------|---------------|
| `TestIBKR_Connect_ValidSession` | Mock `/sso/validate` → 200 + session token |
| `TestIBKR_SubmitOrder_Market` | Mock POST `/iserver/account/{id}/orders` → 200 + order id |
| `TestIBKR_SubmitOrder_Limit` | Verify `orderType: LMT` + `limitPrice` in request body |
| `TestIBKR_SubmitOrder_Stop` | Verify `orderType: STP` + `stopPrice` in request body |
| `TestIBKR_CancelOrder` | Mock DELETE → 200 |
| `TestIBKR_GetOrders_Parse` | Mock GET → JSON → verify Order fields mapped correctly |
| `TestIBKR_GetPositions_Parse` | Mock GET → JSON → verify Position fields |
| `TestIBKR_GetAccount_Parse` | Mock GET → JSON → verify AccountInfo fields |
| `TestIBKR_Connect_Fail` | Mock 401 → verify disconnected |
| `TestIBKR_Session_Refresh` | Verify 4min refresh goroutine fires |
| `TestIBKR_OrderStatus_Mapping` | All IBKR status strings → correct system status |

## Acceptance Criteria

- [ ] IBKR Broker 实现了 `trading.Broker` 接口的全部方法
- [ ] `Connect()` 成功建立 IB Gateway session，失败返回可读错误
- [ ] `SubmitOrder` 支持 market / limit / stop 三种订单类型
- [ ] 订单状态映射正确（IBKR `Filled` → 系统 `filled` 等）
- [ ] Session 后台每 4 分钟自动刷新
- [ ] 应用启动时如果配置了 IBKR 账号，自动注册到 OMS
- [ ] 与 Alpaca Broker 共存，用户可切换券商
- [ ] 所有测试通过，mock HTTP 覆盖主流程 + 错误路径

## Risks / Trade-offs

- **IB Gateway 前置依赖**：用户需手动启动 IB Gateway / Client Portal，不能从 QuantFlow 内一键启动
- **无实时推送**：CP REST API 不支持 WebSocket 订单状态推送，通过轮询 `GetOrders` 每 5 秒刷新
- **自签名证书**：需要 `InsecureSkipVerify: true`，有 MITM 风险（但在 localhost 场景可接受）
- **非专业账户限制**：IB Gateway 对非专业账户有每日 15 分钟交易量限制
- **A 股不可用**：IBKR 不交易 A 股，不影响（QuantFlow 通过 Alpaca/Futu 覆盖 A 股）
