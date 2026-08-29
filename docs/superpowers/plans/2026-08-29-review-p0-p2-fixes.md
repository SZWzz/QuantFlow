# Plan: P0–P2 全面评审修复批次

Spec: [2026-08-29-review-p0-p2-fixes.md](../../specs/2026-08-29-review-p0-p2-fixes.md)

## Task 1 — 仓库卫生

1. `git rm --cached mcp panel-shell-task-1`
2. `.gitignore` 追加：
   ```
   /mcp
   /panel-shell-task-*
   /.venv/
   /python/.venv/
   ```
3. 检查 19 个 `.claude/worktrees/*` 是否有未提交改动；无改动的删除，`git worktree prune`
4. 验证：`git ls-files | grep -E "^mcp$|panel-shell"` 无输出

## Task 2 — AGENTS.md 入库并修正

1. 目录结构一节：`app/` → 仓库根（`main.go`、`app*.go`、`internal/`）；面板数 50+ → 87
2. `git add AGENTS.md`

## Task 3 — 版本统一 2026.8.29

1. `frontend/package.json` version → `2026.8.29`
2. `README.md` badge `动态-2026.7.25` → `动态-2026.8.29`
3. CHANGELOG 新条目 `## [2026.8.29] - 2026-08-29`（本批次全部完成后填写内容）

## Task 4 — TS 错误修复（121 → 0）

1. 导出完整错误清单：`npx vue-tsc --noEmit 2>&1 | grep "error TS" > /tmp/ts-errors.txt`
2. 分类处理：
   - `Property X does not exist on type 'WailsApp'` → 在 `frontend/src/lib/wails.ts` 接口补齐（先确认 Go 端 `app*.go` 有同名导出方法，签名以 Go 返回结构为准）
   - 字段缺失（如 `QuoteData.turnover`）→ 以 Go struct json tag 为准补字段
   - `Promise<...> | undefined` → 修调用方判空
3. 每修一个文件跑 `vue-tsc --noEmit` 计数下降；最后 `npx vitest run` 回归
4. 禁止 `as any` / `@ts-ignore`

## Task 5 — SettingsPanel 导出 stub

1. 读 `internal/data/exporter.go` 与 `app_data.go` 确认导出能力
2. 有 → 面板接真实导出（`await confirmDialog`/`alertDialog` 替换 `alert`）；无 → 隐藏入口

## Task 6 — Python venv + pytest

1. `cd python && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt pytest pytest-asyncio`
2. `.venv/bin/python -m pytest tests/ -q`，记录结果；失败项如实报告

## Task 7 — 覆盖率 backtest/storage ≥ 60%

1. `go test -coverprofile` 定位未覆盖函数
2. backtest：补 price_limit 边界、CN 引擎 T+1/涨跌停/印花税用例
3. storage：补迁移顺序、CRUD 往返用例
4. 只加测试不改实现

## Task 8 — CHANGELOG + 提交

1. CHANGELOG 填写本批次 Added/Changed/Fixed
2. 提交（先确认版本日期 = 2026.8.29）

## Task 9 — 合并 main

1. `git checkout main && git merge --no-ff feat/frontend-visual-unification`
2. 合并后全量验证；冲突则逐一解决，不 push

## Task 10 — 最终全量验证

```bash
go vet ./... && go test ./... -count=1
cd frontend && npx vue-tsc --noEmit && npx vitest run
cd python && .venv/bin/python -m pytest tests/ -q
```
