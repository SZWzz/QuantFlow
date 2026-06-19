# [待开发/部分完成] Broker 补齐 + 交易增强

> **Status**: PENDING — 后续开发
> **Proposal ref**: NEW_PROJECT_PROPOSAL.md §4.1.2 (交易执行), §9 路线图 2.3/4.3
> **Priority**: 🟡 中

## Motivation

当前只有 Futu + Binance 两个 broker adapter。规划覆盖 A 股/港股/美股/加密 四大市场的核心券商。

## 缺失 Broker 适配器

| Broker | 市场 | 优先级 | 说明 |
|--------|------|--------|------|
| **Alpaca** | 美股 | 🔴 | REST + WebSocket，有开放 API，最适合美股 Paper/Live |
| **IBKR (盈透)** | 全球 | 🔴 | 覆盖面最广，IB Gateway/API 复杂 |
| **OKX** | 加密 | 🟡 | 永续合约+现货 |
| **长桥 (LongPort)** | 港股/美股 | 🟡 | 港美股券商 |
| **Bybit** | 加密 | 🟢 | 永续合约 |
| **华泰 (HTSC)** | A股 | 🟢 | A 股券商（需要 Python 通达信桥接） |

## 现有 Broker 待补全

### Futu (已完成基础框架)
- [ ] 实盘下单支持
- [ ] WebSocket 行情订阅
- [ ] 账户/持仓查询

### Binance (已完成基础框架)
- [ ] 永续合约交易
- [ ] WebSocket 实时成交推送
- [ ] 资金费率监控

## 交易增强组件

| 组件 | 状态 | 说明 |
|------|------|------|
| AccountManager | 📋 | 多券商账户统一管理 |
| ActionCenter (审批流) | 📋 | 交易审批/风控审核 |
| WebhookListener | 📋 | 外部信号→交易 |
| AlgoEngine (TWAP/VWAP) | 📋 | 算法执行引擎 |
| Broker切换面板 | 📋 | BrokerStatusPanel |

## 实现模式

```go
// internal/trading/brokers/xxx.go
type XxxBroker struct {
    // 实现 trading.Broker 接口
}

func (b *XxxBroker) PlaceOrder(ctx context.Context, order *trading.Order) (*trading.OrderResult, error) { ... }
func (b *XxxBroker) CancelOrder(ctx context.Context, orderID string) error { ... }
func (b *XxxBroker) GetPositions(ctx context.Context) ([]trading.Position, error) { ... }
func (b *XxxBroker) GetOrders(ctx context.Context, filter trading.OrderFilter) ([]trading.Order, error) { ... }
```

## Acceptance Criteria

- [ ] 每个 Broker 实现 `trading.Broker` 接口全部方法
- [ ] 有 mock/unit 测试（不依赖真实 API）
- [ ] BrokerConfig 面板可添加/删除/测试连接
- [ ] OrderEntry 面板可选择目标 Broker
- [ ] 现有测试通过

## 工作量估算

- Alpaca: ~3 天
- IBKR: ~5 天
- OKX: ~2 天
- 长桥: ~2 天
- Bybit: ~1 天
- 华泰: ~3 天 (需 Python 桥接)
- 交易增强 (AccountManager/ActionCenter): ~3 天
- 测试: ~3 天
- **合计: ~22 天**

## Risks / Trade-offs

- IBKR API 文档复杂，gateway 需单独进程
- 华泰等 A 股券商需要 Python 通达信/同花顺桥接
- 多 Broker 账户管理涉及 AES-256 API Key 加密存储
