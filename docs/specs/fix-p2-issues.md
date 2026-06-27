# P2 边界优化修复 Spec

> **版本**: 0.2.0 | **日期**: 2026-06-27 | **状态**: Draft
> **基于**: P0/P1 已完成修复，本 Spec 覆盖剩余 P2 边界/规范/可维护性问题

## P2-1: 方差样本估计 (N-1)

**问题**: `metrics.go:58` 和 `risk.go:69` 用 `variance / float64(nDays)` (总体方差)
**修复**: 当 nDays > 1 时用 `variance / float64(nDays-1)` (样本方差)
**文件**: `internal/backtest/metrics.go:58`, `internal/portfolio/risk.go:69`

## P2-2: VaR 符号约定

**问题**: `risk.go:100` VaR 报告为负值，行业惯例为正数（损失金额）
**修复**: `VaR95: abs(var95) * totalValue`
**文件**: `internal/portfolio/risk.go:100`

## P2-3: ST 股 ±5% 涨跌停

**问题**: `price_limit.go:10-11` 注释说 "deferred to Phase B"
**修复**: 在 `PriceLimitFor` 函数中用 symbol 前缀判断 ST（A股 ST/*ST 为 5%），覆盖 symbol 含 "ST" 的股票
**文件**: `internal/backtest/price_limit.go`, `internal/trading/oms.go:352-366`

## P2-4: 订单 ID 碰撞风险

**问题**: `oms.go:110` `uuid.New().String()[:8]` 只取 8 字符，碰撞概率高
**修复**: 保留完整 uuid 或取前 12 字符
**文件**: `internal/trading/oms.go:110, 214`

## P2-5: 印花税率按日期可配

**问题**: `engine_cn.go:88` `stampDuty` 写死 0.0005
**修复**: 加一个 StampDutyRate 字段到 CNEngine config，默认 0.0005。提供历史税率映射（2008 年前 0.003, 2008-2023.8 期间 0.001, 2023.8.28 后 0.0005）
**文件**: `internal/backtest/engine_cn.go:87-89`, `engine_cn.go:60-84`

## P2-6: ECharts 按需导入

**问题**: 面板用 `import * as echarts` 全量导入 ~900KB
**修复**: 面板只导入使用的 chart 类型和组件
**文件**: `frontend/src/terminal/panels/CandlestickPanel.vue:8`, 等

## P2-7: tsconfig 严格模式

**问题**: `strict: true` 未开启
**修复**: 开启 strict，修复产生的 type errors
**文件**: `frontend/tsconfig.json`

## P2-8: RLPredict 桩函数

**问题**: `ml/engine.py:208-220` 忽略 model_id，永远返回 hold
**修复**: 调用实际模型推理
**文件**: `python/src/ml/engine.py:208-220`

## P2-9: DQN sharpe 除零保护

**问题**: `rl/algorithms/dqn.py:94` 单元素 data 导致 std=0，sharpe 返回天文数字
**修复**: 加 `if len(data) < 2 or std == 0` 保护
**文件**: `python/src/ml/rl/algorithms/dqn.py:94`

## P2-10: Anthropic finish_reason 未映射

**问题**: `anthropic_provider.py:157` finish_reason 未正确映射，前端解析错乱
**修复**: 映射 `end_turn → stop, max_tokens → length`
**文件**: `python/src/llm/anthropic_provider.py:157`

## P2-11: 加密永续合约 stub

**问题**: binance 仅支持现货，无 USDⓈ-M 合约接口
**修复**: 新建 `internal/market/adapters/binance_futures.go`，支持 `/fapi/v1` 接口获取 perp 行情
**文件**: 新建 `internal/market/adapters/binance_futures.go`
