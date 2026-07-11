# P2 — 能力升级：实时推送、面板优化与实盘验证

> 日期: 2026-07-11 | 优先级: P2 | 预计工作量: 7-9 天

## Motivation

P0/P1 解决「正确性」问题后，P2 聚焦「能力升级」——让产品从"能用"走向"好用"：

1. **加密市场是真 WebSocket 的唯一落地场景** — A 股/港股数据源不存在公开 WS，加密是唯一能实现 <100ms 推模式的市场，也是验证架构的最佳切入点
2. **93 个面板同时活跃** — 内存和 WebSocket 连接线性增长，不可见面板需要自动休眠
3. **券商适配器已写好但从未实盘验证** — Alpaca/IBKR 的 paper trading 端到端闭环是证明系统可靠性的关键里程碑

## Design

### Part A: 加密市场真 WebSocket 推送

**目标架构**:
```
Binance/OKX/Gate WSS → wsconn.Manager → marketHub.Publish() → wsHub.Broadcast() → 前端
  (交易所原生WS)     (连接池+重连)     (内存发布)          (内部WS推送)       (浏览器)
```

**设计决策**:

1. **新建 `internal/market/wsconn/` 包** — 交易所 WebSocket 连接管理器
   - `Manager` 管理多个交易所的 WS 连接
   - 共享重连逻辑（指数退避: 1s→2s→4s→8s→max 30s）
   - 共享 ping/pong 心跳
   - 共享订阅管理（按 topic 去重）

2. **适配器增加可选接口** — 不修改 `Adapter` 接口
   ```go
   type WSConnector interface {
       ConnectWS(ctx context.Context, hub *MarketDataHub) error
       SupportsWS() bool
   }
   ```
   - `BinanceAdapter` 实现 `WSConnector`，连接 `wss://stream.binance.com:9443/ws`
   - `OKXAdapter` 实现 `WSConnector`，连接 `wss://ws.okx.com:8443/ws/v5/public`
   - `GateAdapter` 实现 `WSConnector`，连接 `wss://api.gateio.ws/ws/v4/`

3. **降级策略**: WS 活跃时走推模式，断连自动回退到 HTTP 5s 轮询（现有 `QuotePoller`）

4. **不修改 CN/US/HK 的 HTTP 轮询链** — 并行运行，互不影响

### Part B: 面板虚拟化

**目标**: 不可见面板自动取消 WebSocket 订阅，从 ~93 活跃连接降到 ~10-15 个。

**设计**:

1. **DockTab 增加可见性检测** — 使用 `IntersectionObserver` 或 `v-if` 控制
2. **面板生命周期 hook** — 每个面板实现 `onHidden()` / `onVisible()` 回调
   ```typescript
   // 面板基约定（非强制接口，约定优于配置）
   // onHidden: 取消 WS 订阅、暂停轮询
   // onVisible: 恢复 WS 订阅、启动轮询
   ```
3. **TerminalStore 跟踪活跃面板** — `activePanelIds: Set<string>`，只有可见 tab 的面板在其中
4. **Store 层自动管理** — `useDataStore` 的 `subscribe()` 在面板隐藏时自动降级为缓存模式

### Part C: 实盘 Paper Trading 验证

**目标**: Alpaca Paper Account → 下单 → 成交回报 → 持仓更新 端到端闭环。

**设计**:

1. **Alpaca Paper Account 环境配置**
   - 在 Settings 面板中新增 Alpaca 配置项（API Key + Secret + Paper/Live 切换）
   - `BrokerConfig` 结构体支持 `Environment` 字段（paper/live）

2. **端到端测试场景**
   - 连接 → 获取账户 → 获取持仓 → 下市价单 → 等待成交 → 获取订单状态 → 取消未成交单
   - 全流程在 `brokers/alpaca_test.go` 中以 `//go:build integration` tag 实现

3. **错误处理增强**
   - 网络超时重试（已有 retry 逻辑，确认在 broker 层生效）
   - 订单拒绝原因展示（资金不足、停牌、T+1 限制等）

## Acceptance Criteria

### Part A: 加密 WS
- [ ] `wsconn.Manager` 支持 Binance/OKX/Gate 三个交易所的 WS 连接
- [ ] WS 活跃时，加密行情延迟从 5s 降到 <100ms（timestamp 差值验证）
- [ ] WS 断开后自动回退到 HTTP 5s 轮询，重连后恢复 WS
- [ ] 10 分钟压测无内存泄漏（goroutine 数稳定）
- [ ] CN/US/HK 的行情不受影响，仍走 HTTP 轮询

### Part B: 面板虚拟化
- [ ] 切换到非活跃 tab 的面板取消 WS 订阅
- [ ] 切换回活跃 tab 后面板恢复 WS 订阅，数据在 1s 内刷新
- [ ] 内存使用量在 20 个面板打开时 < 打开全部 93 个面板时的 30%
- [ ] 所有现有面板测试通过

### Part C: 实盘验证
- [ ] Alpaca Paper Account 连接成功
- [ ] 市价单/限价单下单 → 成交 → 持仓更新全流程通过
- [ ] 断网重连后状态恢复
- [ ] 集成测试 `//go:build integration` 通过

## Risks / Trade-offs

- **加密 WS 需要 API Key 吗**: Binance 公开行情 WS 无需认证，只有账户/订单 WS 需要。公开行情（ticker/kline/depth）免费且无需 key。
- **WS 连接数管理**: Binance 对单 IP 有连接数限制（通常 1024），但当前场景下 3 个交易所最多各 1 个连接，远低于限制。
- **面板虚拟化对 UX 的影响**: 切换 tab 时有 ~500ms 的数据恢复延迟（重新订阅→等待下一个 tick）。对于非高频交易场景可接受。
- **Alpaca Paper 的限制**: Paper API 有一定延迟（模拟撮合），且不模拟所有市场微观结构（无滑点、无部分成交）。
