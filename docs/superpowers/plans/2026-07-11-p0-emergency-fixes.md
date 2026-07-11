# P0 紧急修复 — 实施计划

> 日期: 2026-07-11 | 参考 Spec: `docs/specs/2026-07-11-p0-emergency-fixes.md`

## 任务分解

### Task 1: 重命名 SquareRootSlippage → QuadraticSlippage

**文件**: `internal/trading/backtest/engine_cn.go`

**步骤**:
1. 将类型名 `SquareRootSlippage` 改为 `QuadraticSlippage`
2. 更新 struct 注释: `// QuadraticSlippage uses quadratic impact model: Base * (1 + impact²)`
3. 更新构造函数注释: `// NewQuadraticSlippage creates a quadratic market impact model`
4. 搜索全项目确认无其他引用: `grep -r "SquareRootSlippage" .`
5. 运行 `go test ./internal/trading/backtest/... -v`

**提交**: `fix(backtest): rename SquareRootSlippage to QuadraticSlippage — name matched quadratic formula`

---

### Task 2: 修复 PDT 交易日计数

**文件**: `internal/trading/backtest/engine_us.go`

**步骤**:
1. 找到 PDT 检测逻辑中 `AddDate(0, 0, -5)` 的位置
2. 改为基于 `runner.bars` 日期数组计算 5 个交易日前的日期
3. 实现辅助函数 `tradingDaysAgo(dates []time.Time, from time.Time, n int) time.Time`
4. 编写测试验证:
   - 周五→周一不算周末
   - 5 个交易日内 ≥4 次 day trade 触发 PDT 限制
   - 5 个交易日内 <4 次不触发
5. 运行 `go test ./internal/trading/backtest/... -v`

**提交**: `fix(backtest): PDT day-trade window uses trading days not calendar days`

---

### Task 3: 注释 Stub 指标节点注册

**文件**: `internal/workflow/nodes/register.go`

**步骤**:
1. 找到 `RegisterAll()` 函数
2. 注释掉 19 个 `register(&IndicatorXXXNode{})` 调用
3. 在注释上方添加说明: `// TODO: 以下 19 个指标节点为 stub，需接通 Python gRPC 后恢复`
4. 运行 `go test ./internal/workflow/... -v` 确认无破坏
5. 验证节点注册数量从 95 降到 76

**提交**: `refactor(workflow): unregister 19 stub indicator nodes — restore after gRPC wiring`

---

### Task 4: 修复 NodePalette 前端测试

**文件**: `frontend/src/workflow/__tests__/NodePalette.test.ts`

**步骤**:
1. 打开测试文件，找到 `should mount without crashing` 测试
2. 在 mount 配置中添加 `$t` 全局 mock
3. 运行 `npx vitest run NodePalette` 确认通过
4. 运行全量测试 `npx vitest run` 确认 198/198 通过

**提交**: `test(frontend): fix NodePalette test — add $t i18n mock`

---

### Task 5: 更新 CHANGELOG

**步骤**:
1. 在 `CHANGELOG.md` 的 `[2026.7.11]` section 添加 4 条 Fixed 记录

**提交**: `chore: update CHANGELOG for P0 emergency fixes`

---

## 执行顺序

```
Task 1 (Slippage) ──┐
Task 2 (PDT)      ──┼── 并行执行（3 个 Go 任务独立）
Task 3 (Stub)      ──┘
Task 4 (Frontend)  ──── 可与上面并行
Task 5 (CHANGELOG) ──── 最后执行
```
