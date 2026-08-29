# P0–P2 全面评审修复批次

## Motivation

2026-08-29 项目全面评审发现以下问题，本批次统一修复：

1. **P0** `npx vue-tsc --noEmit` 失败，共 121 个 TS 错误（集中在 GovDataPanel/MarketScannerPanel/BacktestPanel 等 20+ 面板）。2026-07-30 `AppMethods` 接口类型化后，多个面板调用了接口中不存在的方法（`CheckWashSale`、`SearchSymbols`、`GetWhaleTransactions` 等），类型契约断裂。
2. **P0** `feat/frontend-visual-unification` 领先 main 48 个提交（208 文件，+11.5k/−9.6k），合并债务持续累积。
3. **P1** Python 验证链断裂：仓库无 `.venv`，28 个测试文件无法执行。
4. **P1** 仓库卫生：15.7MB 编译产物 `mcp` 与任务草稿 `panel-shell-task-1` 被 git 追踪；`.claude/worktrees/` 残留 19 个 worktree 共 2.3GB。
5. **P2** 版本日期不合规且三处不一致（package.json 2026.7.30 / README badge 2026.7.25 / 今日 2026-08-29）。
6. **P2** `AGENTS.md` 未被 git 追踪，且目录结构描述与现实脱节（Go 代码在仓库根而非 `app/`；面板实为 87 个）。
7. **P2** `SettingsPanel.vue:134` 存在面向用户的 stub：`alert('Export data stub — not yet implemented')`，且使用被禁用的 `alert()`。
8. **P2** 核心包覆盖率不足：backtest 45.7%、storage 38.9%（目标 60%+）。

## Design

### 1. TS 错误修复（不改运行时行为）

- 对 `WailsApp` 接口缺失的方法：确认 Go 端 `app*.go` 存在对应导出方法后，在 `frontend/src/lib/wails.ts` 补齐类型化声明与包装函数。
- 对面板内类型不匹配（如 `Promise<[QuoteData, string]> | undefined`、缺少 `turnover` 字段）：修正调用方类型或在数据类型上补字段，**以 Go 端真实返回结构为准**。
- 禁止用 `as any` / `@ts-ignore` 压错；确实需要宽类型的位置补最小化接口定义。

### 2. 分支合回 main

- main 仅领先 1 个提交，直接 `git checkout main && git merge --no-ff feat/frontend-visual-unification`，本地完成、不 push。
- 合并后跑一次全量验证。

### 3. Python 验证链

- `python -m venv .venv`（项目内），安装 `requirements.txt` + pytest，跑通 `pytest tests/ -x -q`。
- `.venv/` 加入 `.gitignore`。

### 4. 仓库卫生

- `git rm --cached mcp panel-shell-task-1`，两者加入 `.gitignore`。
- 清理 `.claude/worktrees/` 残留并 `git worktree prune`。

### 5. 版本与文档

- `frontend/package.json`、`README.md` badge、`CHANGELOG.md` 统一为 `2026.8.29`。
- `AGENTS.md` 入库；目录结构一节改为现实结构（Go 代码在仓库根）；面板数改为 87。

### 6. SettingsPanel 导出 stub

- 检查 Go 端是否已有导出能力（`app_data.go` / `internal/data/exporter.go` 存在 exporter）。若有，接线上导出；若无，隐藏入口并去掉 `alert()`。

### 7. 覆盖率

- 为 `internal/backtest`（价格限制、CN 引擎规则）与 `internal/storage`（迁移、CRUD）补表驱动测试，目标两包均 ≥ 60%。

## Acceptance Criteria

- [ ] `cd frontend && npx vue-tsc --noEmit` 0 错误
- [ ] `go build ./... && go vet ./... && go test ./...` 全绿
- [ ] `cd frontend && npx vitest run` 全绿
- [ ] `cd python && .venv/bin/python -m pytest tests/ -q` 通过
- [ ] `git ls-files` 不再包含 `mcp`、`panel-shell-task-1`
- [ ] package.json / README badge / CHANGELOG 版本均为 2026.8.29
- [ ] backtest、coverage 均 ≥ 60%
- [ ] main 已合并 feature 分支且全量验证通过

## Risks / Trade-offs

- **TS 修复面大（121 个错误）**：只修类型契约，不改业务逻辑；每修一批跑一次 vue-tsc 与 vitest 防止误伤。
- **合并冲突**：main 仅领先 1 提交，冲突面可控；合并不 push，可随时 `git merge --abort` / reset 回滚。
- **覆盖率补充**：以加测试为主；测试暴露出的实现 bug 单独评审后修复（见下）。
- **worktree 清理**：仅删除不在使用中的 agent worktree；若某 worktree 有未提交改动则保留并报告。

## 执行中发现的实现 Bug（测试驱动暴露，已修复）

1. **HK/US 回测引擎 T+1 锁未清理** — OMS `FillOrder` 对所有市场应用 T+1 锁，但只有 CN 引擎在日期变更时 `ClearT1Lock()`；HK/US 引擎从不清理，导致买入后**一切卖出被静默拒绝**（"T+1 lock: cannot sell"）。HK/US 均为 T+0 可交易（T+2 是现金交收而非卖出锁定），修复为每根 bar 开头清理 T+1 锁。
2. **HK/US 止损/止盈路径幻影成交** — `FillOrder` 错误被忽略，交易记录和 P&L 在成交失败时仍被写入。修复为仅在 `FillOrder` 成功时记录交易。
3. **ExecutionRepo.DeleteBefore 时间格式不匹配** — `created_at` 列用 SQLite `datetime('now')` 默认值（空格分隔），而 cutoff 用 RFC3339（`T` 分隔）；字符串比较中 `' ' < 'T'`，同日 cutoff 会误删新记录。修复为统一使用 SQLite datetime 格式。
4. **Python `predict_time_ms` 截断为 0** — `int()` 截断使亚毫秒预测报告 0。修复为 `max(1, ceil(...))`。
5. **前端 2 处真实运行时错误**（随 TS 修复一并纠正）：`GetHKTradeCalendar` → Go 端实际为 `GetHKTradingCalendar`；`GetFinancialForecast` → `GetForecast`；`GetHKIPOData` → `GetHKIPOCalendar(year)`（且返回键为 `listing` 而非面板期望的 `upcoming`）。
