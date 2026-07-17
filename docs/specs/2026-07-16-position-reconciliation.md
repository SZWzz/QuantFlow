# 持仓同步与对账 (Position Reconciliation)

## Motivation

当前 Broker Adapter 能下单和执行查询，但没有定期持仓同步机制。用户通过 IBKR/Futu/Alpaca 在外部交易后，QuantFlow 的仓位状态与 Broker 实际仓位会不一致。同时 Paper Engine 的虚拟持仓也没有与实际成交对账。

长期运行后，用户可能做出基于错误仓位的交易决策。

## Design

### 核心结构

```go
// internal/trading/reconciliation.go
type ReconciliationReport struct {
    Broker    string
    LocalPositions  []Position      // QuantFlow SQLite 记录
    RemotePositions []Position      // Broker API 查询结果
    Matches    []MatchedPosition    // 双方一致的
    Mismatches []MismatchedPosition // 不一致的
    Orphans    []Position           // Broker 有但本地没有
    Missing    []Position           // 本地有但 Broker 没有
    Timestamp  time.Time
}

type MismatchedPosition struct {
    Symbol       string
    LocalQty     float64
    RemoteQty    float64
    LocalCost    float64
    RemoteCost   float64
    DiffQty      float64
    DiffCost     float64
}
```

### 数据流

```
定时触发 (每 15min 或用户手动)
  → TradingEngine.ReconcileAll()
    → 遍历活跃 Broker
      → broker.GetPositions()      (远程)
      → storage.GetPositions()     (本地)
      → 对比生成 ReconciliationReport
      → 存储报告到 SQLite
      → ws push "trading:reconciliation"
        → 前端状态更新
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/trading/reconciliation.go` | 新建 | 对账引擎 |
| `internal/trading/repo.go` | 修改 | 新增 `SaveReconciliationReport` / `GetReconciliationReports` |
| `internal/storage/migrations/019_reconciliation.sql` | 新建 | `reconciliation_reports` 表 |
| `internal/trading/engine.go` | 修改 | 新增 `ReconcileAll`, `ScheduleReconciliation` |
| `frontend/src/terminal/panels/PositionPanel.vue` | 修改 | 显示对账状态标记 |
| `frontend/src/stores/portfolio.ts` | 修改 | 新增 reconciliation state |

### SQLite Schema

```sql
-- migration 019
CREATE TABLE reconciliation_reports (
    id          TEXT PRIMARY KEY,
    broker      TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,
    report_json TEXT NOT NULL,  -- JSON 序列化
    status      TEXT NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','matched','mismatch','error'))
);

CREATE INDEX idx_recon_broker_time ON reconciliation_reports(broker, timestamp);
```

### 前端展示

```
PositionPanel 表头:

Symbol  │  Qty (Local)  │  Qty (Remote)  │  Diff  │  Cost Basis  │  Actions
─────────────────────────────────────────────────────────────────────────
AAPL    │  100          │  100            │   0    │  $175.20     │  ✅
TSLA    │  50           │  48             │  +2    │  $245.00     │  ⚠️ Sync
BTC     │  0.5          │  0.5            │   0    │  $42,000     │  ✅
```

- `Diff != 0` 的行背景色黄色高亮
- ⚠️ Sync 按钮 → 执行 ResolveMismatch（以下方策略之一）
- 面板顶部显示最后对账时间 + 对账状态

### 不一致处理策略

| 场景 | 自动处理 | 用户操作 |
|------|---------|---------|
| DiffQty > 0 (本地多) | 记录差异，标记需关注 | 手动调整或执行"拉齐"→ 以 Broker 为准覆盖本地 |
| DiffQty < 0 (本地少) | 自动拉取 Broker 补全本地 | 可选忽略 |
| Broker 有但本地无 | 自动添加（标记为"自动发现"） | 确认或删除 |
| 本地有但 Broker 无 | 保留但不计入风控 | 确认为已平仓 |

### 自动调度

- `schedule.Scheduler` 注册 `ReconcileAll` 任务，间隔 15 分钟
- 仅活跃 Broker 执行对账
- 非交易时段跳过

## Acceptance Criteria

- [ ] ReconciliationEngine 能定时查询所有活跃 Broker 持仓并与本地对比
- [ ] 对账报告包含 Matches / Mismatches / Orphans / Missing 四类
- [ ] 报告持久化到 SQLite，可查询历史对账记录
- [ ] PositionPanel 展示对账差异（本地 vs 远程数量 + 成本）
- [ ] 差异行高亮显示，提供 Sync 操作按钮
- [ ] 自动对账每 15 分钟执行一次（仅交易时段）
- [ ] 对账完成通过 ws 推送通知前端
- [ ] Go 测试覆盖：mock broker 返回差异仓位 → 生成正确报告
- [ ] 迁移 SQL 向前兼容

## Risks / Trade-offs

- **风险**: Broker API 限流（如 Alpaca 每分钟 200 次）。→ 对账频率间隔 15min，不与其他 API 调用竞争
- **风险**: 对账期间新成交导致 race。→ 对账快照时刻一致（先锁再查）
- **Trade-off**: 不自劢修正差异，只标记 + 用户确认手动修复。自动修改风险太大
