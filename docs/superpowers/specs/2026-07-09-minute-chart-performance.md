# 分时图性能优化

## Motivation

分时图（分时线）在以下场景有明显卡顿/延迟：
- **初次加载**：切换到分时 tab 后，需要等待 2-5 秒才出图（背后走 mootdx gRPC）
- **盘中刷新**：每 5 秒轮询时，即使没有新数据到达，ECharts 也会全量重建 option → 全量 diff → 无意义渲染
- **非 A 股市场**：港股/美股/加密也发起轮询并失败，浪费 IPC 和前端资源

## Design

### 数据流对比

**当前：**
```
loadMinuteLine() → minuteTicks.value = newArray[] → computed minuteChartOption
→ buildMinuteOption() → 全新 ECBasicOption → VChart :option 触发 ECharts 全量 diff
                                                         ↑ 每 5s 执行，数据没变也执行
```

**改进后：**
```
loadMinuteLine() → minuteTicks.value = newArray[]
→ computed minuteChartOption
  → 计算 dataKey = length|lastTime|lastPrice
  → 与缓存 key 比较
    → 相同：return 上次 option 对象引用（跳过 ECharts diff）
    → 不同：buildMinuteOption() → 缓存新 option → return
```

### 组件设计

#### 1. CandlestickPanel.vue

- `minuteTicks` 改为 `shallowRef<MinuteTick[]>` 避免深响应式追踪
- 新增 `minuteOptionCache` 缓存对象（key + option）
- `minuteChartOption` computed 加入缓存守卫
- `startMinutePolling()` 开头加非 CN 早期返回
- 首次加载时展示 `SkeletonPanel`（已有 `minuteLoading`，需绑定模板）
- `sortMinuteTicks` 改为 `Array.sort` 替代手动插入排序

#### 2. buildChartOption.ts

- `buildMinuteOption` 接受 `prevKey: string` 参数，外部可传入上次 dataKey
- 内部在函数开头比较 key，无变化直接 return `null`
- 调用方检测 `null` 时跳过 VChart option 更新

#### 3. KlineChart.vue

- 新增 `dataUpdateKey: string` prop（轻量通知：数据变了但结构没变）
- 当 `dataUpdateKey` 变化时，直接调用 `chartInstance.setOption({ series: [{ data: newData }] })` 而不是完全替换 `option`
- 保持 `:option` prop 用于结构变更（主题/指标切换）

#### 4. useIndicators.ts

- `IndicatorCache` key 从 `minute-{indicator}-{ticks.length}-{bottomMode}` 扩展为 `minute-{indicator}-{ticks.length}-{lastPrice}-{bottomMode}`
- 避免价格上涨但数量不变时的缓存误命中

### 边界情况

| 场景 | 行为 |
|------|------|
| 非 CN 市场 | 分时 tab 按钮变灰/隐藏，不启动轮询 |
| 首次加载（SQLite hit） | ~50ms 出图，骨架屏几乎不可见 |
| 首次加载（SQLite miss + mootdx） | 骨架屏展示，2-5s 后出图 |
| 盘中无新 tick | computed 返回缓存 option，ECharts 0 开销 |
| 盘中新 tick 到达 | 仅更新 `series[i].data`，不做结构 diff |
| 切换指标（volume→MACD） | 更新 dataUpdateKey，触发结构变更 |
| 切换 symbol | 清空缓存，全新加载 |
| 收盘后 | 不启动轮询，使用最近交易日缓存 |

### 风险 / Trade-offs

| 风险 | 缓解 |
|------|------|
| dataKey 碰撞（不同数据但 key 相同） | key 包含 length + lastTime + lastPrice，同一 symbol 的分时数据不会出现不同数据但三者完全相同的情况 |
| `shallowRef` 导致 Vue 内部更新遗漏 | minuteTicks 只在外层赋值（`minuteTicks.value = [...]`），`shallowRef` 恰好匹配此模式 |
| ECharts 直接操作与 Vue 状态不同步 | 只在 `dataUpdateKey` 变化时用 `setOption`，结构变更走正常 `:option` prop，两者互斥 |

## Acceptance Criteria

- [ ] 初次打开分时图，SQLite 缓存命中时 100ms 内展示
- [ ] 盘中无新 tick 时，computed 跳过后端 option 构建（肉眼无闪烁）
- [ ] 非 CN 市场不启动分时轮询（日志确认）
- [ ] `shallowRef` 不引入 Vue 响应式 bug
- [ ] 全量测试通过：`go test ./...` + `npx vitest run` + `vue-tsc --noEmit`
