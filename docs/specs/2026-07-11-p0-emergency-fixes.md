# P0 — 紧急修复：金融正确性与代码健康

> 日期: 2026-07-11 | 优先级: P0 | 预计工作量: 4-6 小时

## Motivation

评审发现 4 个阻塞性问题，直接影响回测 PnL 准确性、用户体验和 CI 健康：

1. **SquareRootSlippage 名实不符** — 代码实际是二次冲击（`impact²`），不是平方根（`√impact`），回测大单滑点失真
2. **PDT 日交易窗口用日历日** — SEC 规定是 5 个**工作日**，当前 `AddDate(0,0,-5)` 使用日历日
3. **19 个 Stub 指标节点** — 845 行零功能死代码，暴露给用户但不可用
4. **最后 1 个前端测试失败** — NodePalette 测试 `$t` mock 缺失

## Design

### 1. SquareRootSlippage 修复

**文件**: `internal/trading/backtest/engine_cn.go`

**方案 A（推荐）**: 改名为 `QuadraticSlippage`，保持公式不变。
- 理由: 二次冲击模型本身是有效的（大单冲击随交易量平方增长），只是名字错了
- 同时在注释中说明这是二次冲击模型

**方案 B**: 改公式为 `Base * (1 + math.Sqrt(impact))`。
- 理由: 匹配名字的语义

选择方案 A——改名不改公式，因为二次模型在某些场景更保守（大单滑点更高），且生产代码不应因评审而改变有效行为。

### 2. PDT 交易日修复

**文件**: `internal/trading/backtest/engine_us.go`

将 `AddDate(0, 0, -5)` 改为基于交易日计数的逻辑：
- 遍历回测日期数组，往前数 5 个交易日
- 时间窗口内的 day trade 计入 PDT 计数

### 3. Stub 节点处理

**文件**: `internal/workflow/nodes/indicator_*.go` (19 文件)

**方案**: 从 `nodes/register.go` 的 `RegisterAll()` 中注释掉 19 个 stub 指标的注册。
- 不删除源文件（保留代码，后续接通 gRPC 时可恢复）
- 用户不再在面板中看到不可用的节点

### 4. 前端测试修复

**文件**: `frontend/src/workflow/__tests__/NodePalette.test.ts`

在测试的 mount 选项中补充 `$t` mock：
```typescript
global: {
  mocks: {
    $t: (key: string) => key,
  }
}
```

或复用 `mocks.ts` 中的统一 mock。

## Acceptance Criteria

- [ ] `SquareRootSlippage` 重命名为 `QuadraticSlippage`，编译通过，现有测试通过
- [ ] PDT 日交易使用交易日计数，添加测试验证 5 个交易日内 ≥4 次 day trade 触发限制
- [ ] 19 个 stub 节点不再出现在 workflow 节点面板中
- [ ] `npx vitest run` 全部 198 个测试通过
- [ ] `go test ./internal/...` 全部通过（含新增 PDT 测试）

## Risks / Trade-offs

- **Slippage 重命名**: 如果有外部配置引用 `SquareRootSlippage` 字符串，需要同步更新。当前搜索代码库，该类型名仅在 `engine_cn.go` 中使用。
- **PDT 修复**: 交易日计数依赖回测日期数组，需要确保日期数组已排序且无重复。
- **Stub 节点注释**: 不影响已保存的 workflow 文件（它们引用的是 node type string，不是 Go 类型）。如果 workflow 中包含这些节点，执行时会报 "unknown node type"。
- **前端测试**: `$t` mock 可能与 `vue-i18n` 的 `beforeCreate` hook 冲突，需要验证 mock 在 `createPinia` 之前注入。
