# 分时图实时绘制 & 数据持久化 — 设计文档

> 目标：让 CandlestickPanel 的分时图像 同花顺 一样平滑实时绘制，消除"刷新感"，且切 Tab 数据不丢、重启应用可恢复。

## 现状问题

| 问题 | 根因 |
|---|---|
| 每次更新图表"闪一下" | ECharts `setOption` 默认 `notMerge: false` + `animation: true`，收到全量 option 后完整重绘 |
| 切 Tab 回来数据没了 | `minuteTicks` 是组件局部状态，Tab 切换时组件销毁 |
| 10 秒轮询拉全量 240 tick | `GetMinuteLine` 每次返回今日全部数据，而非增量 |
| 无持久化 | 无 SQLite 缓存，重启应用需重新拉取 |

## 目标

- **A — 增量渲染**：ECharts 只追加新数据点，不做全量重绘，消除闪烁
- **B — 数据持久化**：分时数据存入 SQLite，内存 LRU 热缓存，切 Tab/重启可恢复

不做：
- WebSocket 实时推送（保持 10s 轮询）
- 多股同时缓存（初期只缓存当前关注的股票）

---

## 架构

```
┌──────────────┐    增量(最多2个tick)    ┌──────────┐    ┌───────────┐
│ Candlestick  │ ◄────────────────────── │ Go App   │    │ SQLite    │
│ Panel.vue    │   GetMinuteLine(symbol, │ (Wails)  │───▶│ minute_   │
│              │    sinceTimestamp)      │          │    │ cache 表  │
│  provide/    │                        │ Minuting │◄───│           │
│  inject      │                        │ Cache    │    └───────────┘
│  共享 Map    │                        │ (LRU)    │
└──────────────┘                        └──────────┘
```

---

## 实现要点

### 1. Go 后端 — `MinuteCache` 服务

**文件**：`internal/market/minute_cache.go`

#### SQLite 表

```sql
CREATE TABLE IF NOT EXISTS minute_cache (
    symbol    TEXT    NOT NULL,
    date      TEXT    NOT NULL,  -- '2026-06-26'
    tick_time TEXT    NOT NULL,  -- '09:30'
    price     REAL    NOT NULL,
    volume    REAL    NOT NULL,
    avg_price REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, date, tick_time)
);
CREATE INDEX IF NOT EXISTS idx_minute_sym_date ON minute_cache(symbol, date);
```

#### Go 结构体

```go
type MinuteCache struct {
    db   *sql.DB
    lru  *lru.Cache[string, []MinuteTick]  // key: "symbol:date"
    mu   sync.RWMutex
}

type MinuteTick struct {
    Time     string
    Price    float64
    Volume   float64
    AvgPrice float64
}
```

#### 核心方法

| 方法 | 功能 |
|---|---|
| `GetIncremental(symbol, sinceTimestamp)` | since=0 返回全量 + 写 SQLite；since>0 只返回新 tick |
| `loadFromDB(symbol, date)` | SQLite → LRU |
| `saveToDB(tick)` | 单条 INSERT OR IGNORE，批量提交 |
| `NewMinuteCache(db)` | 初始化 LRU(500 entries)，建表 |

#### 接口变更

`app.go` 中 `GetMinuteLine(symbol)` → `GetMinuteLine(symbol string, sinceTimestamp int64)`

### 2. 前端 — ECharts 增量渲染

**文件**：`frontend/src/terminal/panels/CandlestickPanel.vue`

#### 关键改动

1. **`loadMinuteLine(symbol)` 改为增量调用**
   - 首次：`sinceTimestamp = 0`，取全量
   - 后续：`sinceTimestamp = lastTickTime`，只取增量
   - 将增量 tick append 到 `minuteTicks` Map

2. **ECharts 静默更新**
   ```typescript
   chart.setOption(option, {
     notMerge: false,    // 合并模式，不重建
     lazyUpdate: true,   // 延迟更新
     silent: true,       // 不触发事件
   })
   ```

3. **关闭动画**
   ```typescript
   animation: false,
   animationDurationUpdate: 0,
   ```

4. **数据不随组件销毁**
   - 使用 `provide('minuteDataCache', reactiveMap)` 在父级 DockView 提供
   - CandlestickPanel 通过 `inject` 获取，组件卸载时数据保留
   - key = `symbol:date`，同天切回同一股票直接渲染

#### 轮询逻辑调整

```
onMounted:
  loadMinuteLine(symbol, 0)          // 首调全量
  start timer(10s):
    loadMinuteLine(symbol, lastTickTs) // 增量
```

---

## 数据流

```
启动/切股票:
  Frontend → GetMinuteLine("600519", 0)
    → MinuteCache: LRU miss → SQLite 有? 回填 LRU → 返回 []tick
    → SQLite 无? mootdx 拉全量 → 写 SQLite → 回填 LRU → 返回 []tick
  Frontend → Map.set("600519:2026-06-26", ticks) → ECharts 全量渲染(无动画)

每 10s:
  Frontend → GetMinuteLine("600519", lastTickTs)
    → MinuteCache: LRU hit → 过滤 since 之后的 tick
    → 0 tick → 返回空 → 前端跳过渲染
    → N tick → 返回增量 + 写 SQLite
  Frontend → Map 追加 → ECharts appendData 或 setOption(notMerge:false)

切 Tab:
  组件卸载 → 数据在 provide Map 中 → 组件重建 → inject 取回 → 直接渲染

重启应用:
  CandlestickPanel mount → GetMinuteLine(symbol, 0)
    → SQLite 有数据 → LRU 回填 → 返回全量 → 渲染（无需等网络）
```

---

## 边界情况

| 场景 | 处理 |
|---|---|
| 交易日切换 | LRU key 含 date，新一天自动新建，旧 key 靠 LRU 淘汰 |
| SQLite 写入失败 | 仅打 warn 日志，不阻塞前端更新 |
| 非交易时段 | 前端 stop timer，不做轮询 |
| mootdx 拉取失败 | 返回错误，前端保持现状不覆盖已有数据 |
| LRU 满 | 淘汰最久未访问的 key，不影响 SQLite 持久数据 |

---

## 改动清单

| 文件 | 操作 | 说明 |
|---|---|---|
| `internal/market/minute_cache.go` | **新增** | MinuteCache 服务 |
| `internal/market/minute_cache_test.go` | **新增** | 单元测试 |
| `app.go` | **修改** | GetMinuteLine 加 sinceTimestamp 参数 |
| `frontend/src/terminal/panels/CandlestickPanel.vue` | **修改** | 增量渲染 + provide/inject |
| `frontend/src/terminal/DockView/types.ts` | **修改** | 添加 minuteDataCache 类型 |
| `data/quantflow.db` | **自动** | SQLite 新增 minute_cache 表 |

---

## 验证标准

1. 分时图更新无闪烁（肉眼不可见重绘）
2. 切到其他 Tab 再切回，分时图数据不丢失、无重新加载
3. 关闭应用 → 重新打开 → 同一个股票分时图立即显示历史数据
4. 非交易时段不发起网络请求
5. SQLite 写入不影响界面响应（< 5ms 延迟）
