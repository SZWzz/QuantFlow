# 启动时间优化 (Startup Optimization)

## Motivation

当前 QuantFlow 启动顺序是：Go 初始化 → 启动 Wails → Vue mount → 数据加载 → Python sidecar 启动。Python sidecar 可能阻塞启动（gRPC 连接超时），SQLite 数据库随迁移增长变慢，前端 74 个面板全部注册有性能开销。

目标：冷启动 < 2s（从双击到看到终端），热启动 < 1s。

## Design

### 启动时序优化

```
Before:
  main.go init() → 加载所有适配器 → 打开 SQLite → 跑迁移 → 启动 Python → Wails Run
  → Vue mount → 注册 74 面板 → render → 数据加载
                                                      └─ 总计 ~3-5s

After:
  1️⃣ 启动 Wails 窗口 (最快 200ms)
  2️⃣ Vue mount → 骨架屏 → 异步注册面板
  3️⃣ Go 后台并行:
      ├─ SQLite 打开 + 迁移 (goroutine 1)
      ├─ 市场适配器按需加载 (goroutine 2)
      └─ Python sidecar lazy start (goroutine 3, 超时 5s)
  4️⃣ 数据加载延迟到面板激活后
```

### 具体优化措施

#### 1. Python sidecar 懒启动

当前 `main.go` 或 `app_startup.go` 可能等待 Python sidecar 就绪才展示窗口。

修改：Python sidecar 改为 goroutine 异步启动，Wails 窗口立即展示。面板调用 Python 功能时若 sidecar 未就绪，显示 "Python sidecar 加载中…" 状态。

```go
// app_startup.go
func (a *App) startup() {
    // 1. 立即打开窗口
    a.window.Show()

    // 2. 后台初始化
    go func() {
        a.initSQLite()
        a.initMarketHub()
        a.initTradingEngine()

        // 3. Python lazy start (不阻塞)
        go func() {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            if err := a.pythonBridge.Start(ctx); err != nil {
                slog.Warn("Python sidecar unavailable", "err", err)
                // 不阻止应用启动
            }
        }()
    }()
}
```

#### 2. SQLite 迁移优化

SQLite 迁移从串行执行改为只在 schema 变更时执行。每次启动检查 schema version 即可。

```go
// storage/migrate.go
func (s *Storage) Migrate() error {
    currentVersion := s.getSchemaVersion()
    if currentVersion == latestVersion {
        return nil  // 无迁移，1ms 返回
    }
    // 执行增量迁移
}
```

#### 3. 前端面板懒加载

当前所有面板在 `registry.ts` 中静态 import，构建时全部打包。

修改为动态 import（Vite 自动代码分割）：

```typescript
// registry.ts — 懒加载
const panelModules = {
  WatchlistPanel: () => import('./WatchlistPanel.vue'),
  CandlestickPanel: () => import('./CandlestickPanel.vue'),
  // ... 全部 74 个
}
```

用户首次打开面板时加载对应 chunk，不是启动时全加载。

已部分实现（`2026-07-12` 的 CHANGELOG 提到 SkeletonPanel 和 `requestIdleCallback` 预加载 ECharts），需确认覆盖所有面板。

#### 4. 数据源按需初始化

当前市场 Hub 启动时初始化所有适配器（40+ HTTP 客户端 + WebSocket 连接）。

改为：只初始化用户已配置数据源对应的适配器。用户只配了 A 股 → 不加载 Yahoo/Finnhub/Binance 等适配器。

```go
// market/hub.go
func (h *Hub) Init(activeMarkets []string) {
    for _, m := range activeMarkets {
        switch m {
        case "CN":
            h.initAdapters("tencent", "sina", "eastmoney")
        case "HK":
            h.initAdapters("tencent", "yahoo")
        case "US":
            h.initAdapters("yahoo", "finnhub")
        case "CRYPTO":
            h.initAdapters("binance", "okx")
        }
    }
}
```

#### 5. 测量基线

新增启动性能数据收集：

```go
// app_startup.go
type StartupMetrics struct {
    TotalMs      int64
    SQLiteMs     int64
    MarketHubMs  int64
    TradingMs    int64
    PythonMs     int64
    FrontendMs   int64  // Wails 前端加载时间
}
```

记录到日志（首次启动的 startup_metrics.json），帮助持续优化。

### 用户感知优化

| 阶段 | 用户看到什么 | 时间目标 |
|------|------------|---------|
| 双击图标 | Dock 图标弹跳 | 0ms |
| 200ms | 窗口出现 + 骨架屏（终端框架 + 加载动画） | <200ms |
| 500ms | 基础 UI 可交互（菜单栏、状态栏） | <500ms |
| 1s | 面板陆续填充数据 | <1s |
| 2s | 全部面板数据就绪（无 Python 功能） | <2s |
| 5s | Python sidecar 就绪（如有） | <5s |

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/startup/optimizer.go` | 新建 | 启动时序控制 |
| `internal/startup/metrics.go` | 新建 | 启动性能指标 |
| `app_startup.go` | 修改 | 异步启动 Python sidecar |
| `internal/market/hub.go` | 修改 | 按市场按需加载适配器 |
| `frontend/src/terminal/panels/registry.ts` | 修改 | 动态 import 所有面板 |
| `frontend/src/components/SkeletonScreen.vue` | 修改 | 骨架屏展示 |

## Acceptance Criteria

- [ ] 冷启动（清空缓存）从双击到骨架屏 < 500ms
- [ ] 冷启动到面板数据基本就绪 < 2s
- [ ] Python sidecar 启动不阻塞主进程
- [ ] 未就绪的 Python 功能显示"加载中"状态
- [ ] SQLite 迁移在 schema 未变更时 < 5ms
- [ ] 面板全部改为动态 import（构建产物分 chunk）
- [ ] 适配器按已选市场按需加载
- [ ] 启动性能指标记录到日志
- [ ] GitHub Actions 定期 benchmark（防止回归）

## Risks / Trade-offs

- **风险**: 动态 import 导致首次打开某面板时卡顿（加载 chunk）。→ SkeletonPanel 已实现，加载态占位
- **风险**: Python lazy start 导致用户首次调用 LLM/ML 时等待。→ 显示"首次加载较慢"提示
- **Trade-off**: 不做 Go plugin 动态加载（太复杂），只是启动时条件判断
