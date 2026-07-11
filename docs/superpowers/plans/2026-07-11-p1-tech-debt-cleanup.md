# P1 技术债清理 — 实施计划

> 日期: 2026-07-11 | 参考 Spec: `docs/specs/2026-07-11-p1-tech-debt-cleanup.md`

## 阶段一: Go 后端质量修复 (Task 1-6)

### Task 1: Wash Sale 计算修复

**文件**: `internal/trading/wash_sale.go`

**步骤**:
1. 确认 OMS 使用加权平均成本价 (`pos.AvgPrice`)
2. 修改 wash sale 亏损计算: `loss := originalAvgPrice - salePrice` (当前是 `repurchasePrice - salePrice`)
3. 添加测试用例: 买入@10→卖出@8→买入@7→卖出@9，验证亏损 2 元（非 1 元）
4. 运行 `go test ./internal/trading/... -v -run WashSale`

**提交**: `fix(trading): wash sale loss uses original cost basis, not repurchase price`

---

### Task 2: Stamp Duty 四舍五入

**文件**: `internal/trading/backtest/engine_cn.go`

**步骤**:
1. 找到 `tradeValue * e.stampDutyRate` 行
2. 改为 `math.Round(tradeValue*e.stampDutyRate*100) / 100` (精确到分)
3. 更新注释: "2023.8.28 降税后 0.05%, 仅卖出征收, 四舍五入到分"
4. 添加边界测试: `0.005 → 0.01`, `0.004 → 0.00`
5. 运行 `go test ./internal/trading/backtest/... -v`

**提交**: `fix(backtest): round stamp duty to 0.01 CNY (fen precision)`

---

### Task 3: 美股默认 1 股

**文件**: `internal/trading/backtest/engine_us.go`

**步骤**:
1. 找到 `qty = 100` 默认值
2. 改为 `qty = 1`
3. 运行 `go test ./internal/trading/backtest/... -v`

**提交**: `fix(backtest): US default lot size 1 share (not 100)`

---

### Task 4: Sharpe 无风险利率可配置

**文件**: `internal/trading/backtest/metrics.go`, `internal/trading/backtest/engine_cn.go`

**步骤**:
1. 在 `MetricsConfig` 结构体中新增 `RiskFreeRate float64` 字段
2. 默认值 `0.02` (2%)
3. 修改 `SharpeRatio`, `SortinoRatio` 使用 `config.RiskFreeRate` 替代硬编码 `0.02`
4. 更新 CN/US/HK 引擎的 `MetricsConfig` 初始化
5. 添加测试: 验证 `RiskFreeRate=0.03` 时 Sharpe 变化
6. 运行 `go test ./internal/trading/backtest/... -v`

**提交**: `feat(backtest): configurable risk-free rate for Sharpe/Sortino ratios`

---

### Task 5: Channel Leak 修复

**文件**: `internal/market/hub.go`

**步骤**:
1. 新增 `subscriber` struct:
```go
type subscriber struct {
    ch     chan MarketMessage
    closed atomic.Bool
}
```
2. `Subscribe()` 返回的 unsubscribe 函数调用 `sub.close()` → `close(s.ch)` + `s.closed.Store(true)`
3. `Publish()` 中检查 `!sub.closed.Load()` 后再发送
4. 运行 `go test ./internal/market/... -v -run Hub`

**提交**: `fix(market): use subscriber struct to prevent channel leak on unsubscribe`

---

### Task 6: Busy-Wait 修复

**文件**: `internal/workflow/queue.go`

**步骤**:
1. 在 `ExecutionQueue` 结构体中新增 `sync.Cond`
2. 替换忙等 `for { time.Sleep(...) }` 为 `cond.Wait()`
3. Enqueue 时 `cond.Signal()`
4. 运行 `go test ./internal/workflow/... -v -run Queue`
5. 运行竞态检测: `go test -race ./internal/workflow/...`

**提交**: `fix(workflow): replace busy-wait with sync.Cond in execution queue`

---

## 阶段二: 工具函数集中化 (Task 7-8)

### Task 7: 创建 utils.go 并迁移函数

**文件**: 新建 `internal/workflow/nodes/utils.go`，修改 56+ 个引用文件

**步骤**:
1. 创建 `internal/workflow/nodes/utils.go`
2. 从以下文件迁移函数:
   - `macd.go` → `extractFloatSlice`
   - `floatutil.go` → `extractFloat64Slice`（合并入 `extractFloatSlice`，增加类型判断）
   - `factor.go` → `getStringParam`
   - `strategy.go` → `getFloatParam`, `getIntParam`
3. 用 `sed` 批量替换 import:
```bash
# 移除旧 import，更新为新 import
find internal/workflow/nodes -name "*.go" -exec sed -i '' 's/"quantflow\/internal\/workflow\/nodes\/macd"/"quantflow\/internal\/workflow\/nodes\/utils"/g' {} \;
```
4. 逐个文件手动验证，确保只替换了正确的 import
5. 运行 `go build ./internal/workflow/...` 确认编译通过
6. 运行 `go test ./internal/workflow/... -v`

**提交**: `refactor(workflow): centralize utility functions in nodes/utils.go`

---

### Task 8: 合并 extractFloatSlice 和 extractFloat64Slice

**文件**: `internal/workflow/nodes/utils.go`, `internal/workflow/nodes/floatutil.go`

**步骤**:
1. 在 `utils.go` 中实现统一版本，支持 `[]float64` 和 `[]struct{Close float64}` 两种输入
2. 删除 `floatutil.go` 中的 `extractFloat64Slice`，改为调用 `utils.go` 的统一版本
3. 运行全量测试确认兼容性

**提交**: `refactor(workflow): merge extractFloatSlice and extractFloat64Slice`

---

## 阶段三: CandlestickPanel 拆分 (Task 9-13)

### Task 9: 提取 ChartToolbar.vue

**文件**: 新建 `frontend/src/terminal/panels/candlestick/ChartToolbar.vue`

**步骤**:
1. 从 CandlestickPanel.vue 中提取工具栏部分（周期切换、指标选择下拉、导出按钮）
2. 定义 props: `interval`, `indicators`
3. 定义 emits: `update:interval`, `toggle-indicator`, `export`
4. 在 CandlestickPanel.vue 中替换为 `<ChartToolbar>`
5. 运行 `npx vitest run CandlestickPanel`

**提交**: `refactor(frontend): extract ChartToolbar from CandlestickPanel`

---

### Task 10: 提取 KlineChart.vue

**文件**: 新建 `frontend/src/terminal/panels/candlestick/KlineChart.vue`

**步骤**:
1. 提取 ECharts K 线图初始化、配置、dataZoom 处理逻辑
2. 定义 props: `data`, `indicators`, `showVolume`
3. 使用 `useMinuteChart` composable（已存在）
4. 在 CandlestickPanel.vue 中替换

**提交**: `refactor(frontend): extract KlineChart from CandlestickPanel`

---

### Task 11: 提取 IndicatorOverlay.vue

**文件**: 新建 `frontend/src/terminal/panels/candlestick/IndicatorOverlay.vue`

**步骤**:
1. 提取 MA/EMA/Bollinger/MACD/RSI 等指标的计算和渲染逻辑
2. 定义 props: `data`, `activeIndicators`

**提交**: `refactor(frontend): extract IndicatorOverlay from CandlestickPanel`

---

### Task 12: 提取 MinuteChart.vue

**文件**: 新建 `frontend/src/terminal/panels/candlestick/MinuteChart.vue`

**步骤**:
1. 提取分时图部分，复用 `useMinuteChart` composable
2. 定义 props: `symbol`, `market`

**提交**: `refactor(frontend): extract MinuteChart from CandlestickPanel`

---

### Task 13: 验证与收尾

- [ ] CandlestickPanel.vue 缩减到 <400 行
- [ ] 所有子组件单独可测试
- [ ] `npx vitest run` 全部通过
- [ ] 手动验证 K 线图展示正常

**提交**: `refactor(frontend): finalize CandlestickPanel split — 4 sub-components`

---

## 阶段四: CHANGELOG (Task 14)

### Task 14: 更新 CHANGELOG

**提交**: `chore: update CHANGELOG for P1 tech debt cleanup`

---

## 执行顺序

```
阶段一 (Go 后端) ──── 6 个 Task 可并行
    ↓ (等待阶段一全部完成)
阶段二 (工具函数) ── Task 7 → Task 8 (顺序执行)
    ↓ (等待阶段二完成)
阶段三 (前端拆分) ── Task 9 → Task 10 → Task 11 → Task 12 → Task 13 (顺序，有依赖)
    ↓
阶段四 (CHANGELOG) ── Task 14
```
