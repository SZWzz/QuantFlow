# P2 能力升级 — 实施计划

> 日期: 2026-07-11 | 参考 Spec: `docs/specs/2026-07-11-p2-capability-upgrade.md`

## 阶段一: 加密市场真 WebSocket 推送 (3 天)

### Task 1: 创建 wsconn.Manager

**新建文件**: `internal/market/wsconn/manager.go`

**内容**:
```go
// Package wsconn manages external exchange WebSocket connections
// for real-time market data push.
package wsconn

type WSConnector interface {
    ConnectWS(ctx context.Context, hub *MarketDataHub) error
    DisconnectWS() error
    SupportsWS() bool
    ExchangeName() string
}

type Manager struct {
    mu       sync.Mutex
    conns    map[string]*managedConn  // exchange name → connection
    hub      *market.MarketDataHub
    // shared: reconnect backoff, heartbeat, subscription dedup
}
```

**步骤**:
1. 创建 `internal/market/wsconn/` 目录
2. 实现 `Manager` 的核心方法: `Add()`, `Remove()`, `Stop()`
3. 实现共享重连逻辑: 指数退避 1s→2s→4s→8s→max 30s
4. 实现共享心跳: 每 30s ping，60s 无 pong 视为断开
5. 编写 `manager_test.go`

**提交**: `feat(market): add wsconn.Manager for external exchange WebSocket connections`

---

### Task 2: Binance WS 连接器

**新建/修改文件**: `internal/market/adapters/binance_ws.go`

**步骤**:
1. 为 `BinanceAdapter` 新增 `ConnectWS` / `DisconnectWS` / `SupportsWS` / `ExchangeName` 方法
2. 连接 `wss://stream.binance.com:9443/ws`
3. 订阅格式: `{"method":"SUBSCRIBE","params":["btcusdt@ticker","btcusdt@kline_1m"],"id":1}`
4. 解析收到的 JSON → 转换为 `market.QuoteSnapshot` → `hub.Publish()`
5. 编写 `binance_ws_test.go`: mock WebSocket server 验证数据解析正确

**提交**: `feat(market): Binance adapter WebSocket push for real-time quotes`

---

### Task 3: OKX + Gate.io WS 连接器

**新建文件**: `internal/market/adapters/okx_ws.go`, `internal/market/adapters/gateio_ws.go`

**步骤**:
1. OKX: 连接 `wss://ws.okx.com:8443/ws/v5/public`，订阅 tickers/kline channel
2. Gate.io: 连接 `wss://api.gateio.ws/ws/v4/`，订阅 spot.tickers
3. 实现各自的 ConnectWS / DisconnectWS / SupportsWS / ExchangeName
4. 编写测试

**提交**: `feat(market): OKX and Gate.io WebSocket push adapters`

---

### Task 4: 降级策略与集成

**修改文件**: `internal/market/poller.go`, `internal/market/wsconn/manager.go`

**步骤**:
1. `QuotePoller.Poll()` 启动时检查 `wsconn.Manager` 中对应 symbol 是否有活跃 WS
2. 有 WS → 跳过 HTTP 轮询，打日志 `"symbol X covered by WS, skipping poll"`
3. WS 断开 → `Manager` 通知 `QuotePoller` 恢复该 symbol 的轮询
4. 运行集成测试: 启动 WS → 验证 poller 跳过 → 断开 WS → 验证 poller 恢复

**提交**: `feat(market): graceful fallback from WS to HTTP polling`

---

### Task 5: 压测与稳定性

**步骤**:
1. 启动 3 个交易所 WS 连接，持续 10 分钟
2. 监控 goroutine 数: `runtime.NumGoroutine()` 每 30s 输出
3. 监控内存: `runtime.ReadMemStats()` 每 30s 输出
4. 模拟网络断开 5 次，验证重连成功率 100%
5. 确认 CPU 使用率在 WS 模式下 <5%（对比 HTTP 轮询 ~15%）

**提交**: `test(market): WS stability — 10min soak test, reconnect resilience`

---

## 阶段二: 面板虚拟化 (2 天)

### Task 6: DockTab 可见性检测

**修改文件**: `frontend/src/terminal/DockView/DockTab.vue`

**步骤**:
1. 在 DockTab 中新增 `isVisible` ref
2. 监听 tab 切换事件: 当前激活 tab → `isVisible = true`，其他 → `isVisible = false`
3. 通过 `provide/inject` 将 `isVisible` 传递给子面板
4. 确认不影响拖拽/关闭/tearoff 功能
5. 运行 DockView 测试: `npx vitest run DockView`

**提交**: `feat(frontend): DockTab visibility detection for panel lifecycle`

---

### Task 7: 面板生命周期约定

**新建文件**: `frontend/src/lib/composables/usePanelLifecycle.ts`

**内容**:
```typescript
// usePanelLifecycle — composable for panel visibility management
export function usePanelLifecycle(onVisible?: () => void, onHidden?: () => void) {
    const isVisible = inject<Ref<boolean>>('isVisible', ref(true))
    watch(isVisible, (v) => { v ? onVisible?.() : onHidden?.() })
    return { isVisible }
}
```

**步骤**:
1. 创建 composable
2. 在 MarketOverviewPanel、WatchlistPanel、CandlestickPanel 中示范使用
3. 编写 `usePanelLifecycle.test.ts`

**提交**: `feat(frontend): usePanelLifecycle composable for panel visibility management`

---

### Task 8: 热点面板接入生命周期

**文件**: 修改 5-8 个高频使用面板

**步骤**:
1. **WatchlistPanel**: hidden → 取消 `market:quote:*` 订阅, visible → 重新订阅
2. **MarketOverviewPanel**: hidden → 停止 30s 轮询, visible → 恢复
3. **CandlestickPanel**: hidden → 取消分钟线订阅, visible → 恢复
4. **OrderBlotterPanel**: hidden → 停止订单轮询, visible → 恢复
5. 运行所有面板测试确认无破坏

**提交**: `feat(frontend): wire panel lifecycle into hot panels — auto unsubscribe on hide`

---

### Task 9: 虚拟化效果验证

**步骤**:
1. 打开 20 个面板，切换只保留 3 个可见
2. 通过浏览器 DevTools Network 标签确认 WS 消息只发送到可见面板的 topic
3. 对比内存: 虚拟化前 vs 虚拟化后
4. 确认目标: 20 面板打开时 WS 连接数 ≤ 10

**提交**: `perf(frontend): verify panel virtualization — WS connections reduced 70%+`

---

## 阶段三: 实盘 Paper Trading 验证 (2 天)

### Task 10: Alpaca Paper 配置

**修改文件**: `frontend/src/terminal/panels/SettingsPanel.vue`, `internal/trading/brokers/alpaca.go`

**步骤**:
1. SettingsPanel 新增 Alpaca 配置区域: API Key, Secret, Environment (Paper/Live)
2. Go 端 `BrokerConfig` 新增 `Environment string` 字段
3. Alpaca adapter 根据 `Environment` 切换 Base URL (paper-api.alpaca.markets vs api.alpaca.markets)
4. 密钥加密存储（已有 OS keychain 支持）

**提交**: `feat(broker): Alpaca Paper/Live environment switching in settings`

---

### Task 11: 端到端集成测试

**文件**: `internal/trading/brokers/alpaca_integration_test.go`

**步骤**:
1. 编写 `//go:build integration` 测试
2. 测试流程:
   - `TestAlpacaConnect` — 连接验证
   - `TestAlpacaGetAccount` — 获取账户信息
   - `TestAlpacaSubmitMarketOrder` — 下市价单（1 股 AAPL）
   - `TestAlpacaGetOrders` — 查询订单状态
   - `TestAlpacaCancelOrder` — 取消未成交限价单
   - `TestAlpacaGetPositions` — 查询持仓
3. 使用环境变量 `ALPACA_API_KEY` / `ALPACA_SECRET` 配置
4. 测试跳过逻辑: 环境变量缺失时 `t.Skip()`

**提交**: `test(broker): Alpaca paper trading end-to-end integration tests`

---

### Task 12: 错误处理增强

**修改文件**: `internal/trading/brokers/alpaca.go`, `internal/trading/oms.go`

**步骤**:
1. Alpaca API 错误 → 映射为人类可读的中文错误消息
2. 常见错误: 资金不足、停牌、T+1、超出购买力、市场关闭
3. OMS 错误增加 `ErrorCode` 字段
4. 前端展示订单拒绝原因

**提交**: `feat(trading): human-readable order rejection reasons with error codes`

---

## 阶段四: CHANGELOG (Task 13)

**提交**: `chore: update CHANGELOG for P2 capability upgrade`

---

## 执行顺序

```
Task 1 (wsconn.Manager) ──→ Task 2 (Binance WS) ──┬──→ Task 4 (降级+集成) → Task 5 (压测)
                                                   ├──→ Task 3 (OKX+Gate, 可并行)
                                                   
Task 6 (DockTab) → Task 7 (composable) → Task 8 (接入面板) → Task 9 (验证)

Task 10 (Alpaca配置) → Task 11 (集成测试) → Task 12 (错误处理)

三个阶段可并行执行（不同人和不同技术栈）
```
