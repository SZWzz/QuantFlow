# CI/CD 修复与测试稳定性战役

## Motivation

综合评审发现三个阻塞性测试问题，直接影响开发效率和代码质量信心：

1. **CI 配置与 go.mod 版本不一致**：`.github/workflows/ci.yml` 使用 `go-version: '1.22'`，而 `go.mod` 要求 `go 1.25.0`。GitHub Actions 上的 Go 后端 CI 永远失败，CI 形同虚设。

2. **前端测试 23.5% 失败率**：68 测试文件中 22 个失败（48/204 用例）。涉及 stores (data, portfolio) 和大量面板 (ActionCenter, BasketOrder, BrokerStatus, Correlation, CryptoOverview, Distribution, Execution, Geopolitics, GovData, Heatmap, MonteCarlo, OrderBlotter, PredictionMarket, Rebalance, Satellite, Sentiment, SurfaceChart, TickerTape, Watchlist 等)。根因：近期 WebSocket 重构 + OHLCV 变更导致 mock 未同步更新。

3. **Go flaky 测试**：`TestQuotePoller_FetchesAndPublishesData` 时序竞态问题，反映 `market/hub.go` goroutine 同步隐患。

## Design

### 1. 修复 CI 版本配置

**修改文件：**
- `.github/workflows/ci.yml` — go-version: '1.22' → '1.25'
- `.github/workflows/go-test.yml` — 同上（如有）

### 2. 前端测试修复策略

不需要逐面板手动修复，而是系统性方案：

#### 2a. 统一 Mock 层

当前每个面板测试各自 mock Wails IPC，重构后统一到 `frontend/src/__tests__/mocks.ts`：

```typescript
// 统一 mock Wails window.go 对象
export function mockWailsIPC() {
  ;(window as any).go = {
    main: {
      App: {
        SearchSymbols: vi.fn().mockResolvedValue({ data: [] }),
        GetQuote: vi.fn().mockResolvedValue({ data: null }),
        GetMarketOverview: vi.fn().mockResolvedValue({ data: [] }),
        FetchOHLCV: vi.fn().mockResolvedValue({ data: [] }),
        GetMinuteLine: vi.fn().mockResolvedValue({ data: [] }),
        GetIndustryRanks: vi.fn().mockResolvedValue({ data: [] }),
      },
    },
  }
}

// 统一 mock WebSocket
export function mockWebSocket() {
  class MockWebSocket {
    readyState = WebSocket.OPEN
    send = vi.fn()
    close = vi.fn()
  }
  
  vi.stubGlobal('WebSocket', MockWebSocket)
}
```

#### 2b. 自动检测失败模式

```bash
# 按失败原因分组统计
npx vitest run --reporter=json 2>&1 | \
  jq '.testResults[] | select(.status=="failed") | .assertionResults[] | select(.status=="failed") | .failureMessages[0]' | \
  sort | uniq -c | sort -rn
```

预期失败原因分类：
- **TypeError/Wails mock 缺失** (~60%) — 统一 mock 层修复
- **数据格式不匹配** (~25%) — 适配新 store 接口
- **时序问题** (~15%) — 增加 `waitFor` / `flushPromises`

#### 2c. 修复流程

1. 建立 `mocks.ts` 统一 mock
2. `vitest.setup.ts` 全局加载 mock
3. 逐文件运行确认通过
4. CI 中 vitest 加 `--bail=5` 快速失败

### 3. Go flaky 测试修复

**问题代码** `internal/market/poller_test.go:258-298`：

```go
func TestQuotePoller_FetchesAndPublishesData(t *testing.T) {
    // ...
    poller.interval = 10 * time.Millisecond
    // ...
    time.Sleep(60 * time.Millisecond)  // ← 竞态根源：依赖 sleep 等待
    // ...
    msg, ok := marketHub.GetLatest("market:quote:CN:600519")
```

**修复方案：** 用轮询等待替代 sleep：

```go
// 等待最多 500ms 直到数据到达
var msg *MarketMessage
var ok bool
for i := 0; i < 50; i++ {
    msg, ok = marketHub.GetLatest("market:quote:CN:600519")
    if ok {
        break
    }
    time.Sleep(10 * time.Millisecond)
}
```

**修改文件：**
- `internal/market/poller_test.go` — 替换 `time.Sleep(60 * time.Millisecond)` 为轮询等待
- `internal/market/hub.go` — 确认 Publish 的 goroutine-safe（review only）

### 4. CI 增强

在修复后，给 CI 增加防护：

```yaml
jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }     # ← 修复版本
      - run: go build ./...
      - run: go vet ./...
      - run: golangci-lint run ./... --timeout 5m
      - run: go test -race ./... -count=1  # ← 新增 -race
      
  frontend:
    # ...
      - run: cd frontend && npx vitest run --bail=5  # ← 新增 --bail
```

## Acceptance Criteria

- [ ] CI 中 Go 版本为 `1.25`，build/vet/test 全部通过
- [ ] 48 个前端测试全部修复，`npx vitest run` 返回 0
- [ ] Go flaky 测试连续运行 10 次无失败
- [ ] CI 中启用 `-race`，无竞态警告
- [ ] CI 中启用 `--bail=5`，快速失败
- [ ] CHANGELOG 更新

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 统一 mock 可能隐藏真实错误 | 每个 mock 函数用 `vi.fn().mockImplementation(...)` 保留可追踪性 |
| 面板测试修复后发现真实 bug | 修复流程要求先确认测试失败是因为 mock 过时而非真实 bug |
| CI 增加 -race 后变慢 | race detector 约 2-5x 减速，但 CI 总时间仍在可接受范围（<10min） |
