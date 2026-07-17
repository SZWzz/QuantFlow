# 2026-07-16 批次 Plan 执行状态

> 本批次共 15 个 plan（spec 见 `docs/specs/`，plan 见 `docs/superpowers/plans/`），按「基础设施 → 交易核心 → 用户体验 → CI/CD → 安全 → 前端功能」优先级执行。
> 执行方式：subagent-driven-development（每 task 独立子代理实现 + spec/质量双重 review + 全分支终审）。

**最后更新**: 2026-07-17

---

## ✅ 已完成（4/15）

### 1. auto-updater — 自动更新系统

- **Plan**: [2026-07-16-auto-updater.md](./2026-07-16-auto-updater.md)
- **完成日期**: 2026-07-16，7 commits（`50a0f57..d481b6d`）
- **终审结论**: Ready to merge
- **交付**:
  - `internal/updater/` — GitHub Releases 客户端、SHA256 校验、版本比较、三平台 Replace/Restart
  - IPC: `CheckUpdate` / `ApplyUpdate` / `GetUpdateInterval` / `SetUpdateInterval`
  - 前端: UpdatePrompt 对话框 + 启动自动检查 + 设置面板集成
- **已知局限（非阻塞，defer）**:
  - 下载进度 channel 传 nil，UI 无实时进度
  - `fetchChecksum` 不校验 HTTP 状态码（错误消息不清晰但安全）
  - macOS 更新 `/Applications` 需管理员权限

### 2. crash-reporter — 崩溃报告系统

- **Plan**: [2026-07-16-crash-reporter.md](./2026-07-16-crash-reporter.md)
- **完成日期**: 2026-07-17，12 commits（`9ae11bd..aea7511`）
- **终审结论**: Ready to merge（Critical/Important 全部修复）
- **交付**:
  - `internal/crash/` — panic 恢复 + 信号捕获（SIGABRT/SIGSEGV/SIGILL/SIGBUS）、JSON 本地存储、30 天清理、opt-in 上传（默认关闭）
  - `RingBuffer.LastN(100)` 崩溃时嵌入最近日志
  - 前端: CrashDialog（下次启动恢复对话框）+ CrashHistoryPanel（设置 → 崩溃报告）
- **已知局限（defer）**:
  - `crash:detected` Wails 事件 Go 侧未发射（localStorage watermark 是功能路径）
  - 上传端点 `hooks.quantflow.app/crashes` 为占位符
  - AppState getters 部分为 stub（panel_count / workflow_count 恒为 0）

### 3. release-pipeline — GitHub Actions 自动发版流水线

- **Plan**: [2026-07-16-release-pipeline.md](./2026-07-16-release-pipeline.md)
- **完成日期**: 2026-07-17，2 commits（`8ae7647..687abb3`）
- **交付**:
  - `.github/workflows/release.yml` — `git tag v*` push 触发，6 平台并行构建矩阵（macOS arm64/amd64、Linux amd64/arm64、Windows amd64/arm64）
  - `scripts/darwin-package.sh` — macOS .app bundle + DMG 打包
  - `scripts/linux-package.sh` — Linux tar.gz 打包
  - `scripts/windows-package.ps1` — Windows zip 打包
  - `scripts/checksum.sh` — SHA256 checksums 生成
  - `resources/Info.plist.template` — macOS Info.plist 模板
  - Makefile: `release-darwin` / `release-linux` / `release-windows` / `release-checksum` / `release` 目标
  - `scripts/homebrew/quantflow.rb` — Homebrew formula 模板（SHA256 占位符，首次发版后填入实际值）
- **已知局限（defer）**:
  - 代码签名未配置（需 Apple Developer Certificate 在自建 runner 上）
  - .msi 打包延后（WiX Toolset 学习成本）
  - 首次发版需替换 Homebrew formula SHA256 占位符

### 4. error-visibility — 全局错误可见性系统

- **Plan**: [2026-07-16-error-visibility.md](./2026-07-16-error-visibility.md)
- **完成日期**: 2026-07-17
- **交付**:
  - `useToast` composable — 4 类型 toast（info/success/warning/error）、30s 去重合并、自动消失、单例共享状态
  - `ToastContainer.vue` — 固定右上角浮动 toast 容器（slideIn 动画、手动关闭按钮）
  - `StatusBar.vue` — 增强版状态栏：行情源/券商/Python 连接状态行 + 点击弹详情对话框 + 版本号显示
  - `ring_buffer.go` — `SetHub(hub)` 注入 WS Hub，Push 时自动广播日志条目到 `system:notification` topic
  - `GetConnectionStatus()` IPC — 返回实时行情适配器、券商连接（`IsConnected()`）、Python sidecar 三组状态
  - `terminal.ts` — 新增 `connectionStatus` 响应式状态 + `updateConnectionStatus` action
- **已知局限（defer）**:
  - 行情源状态仅检查适配器是否注册，未检查实时可用性（`IsAvailable()`）

---

## ⏳ 未完成（11/15）

### 交易核心（3 个）— 建议下一批执行

| # | Plan | Tasks | 规模 | 说明 |
|---|------|-------|------|------|
| 3 | [daily-pnl-report](./2026-07-16-daily-pnl-report.md) | 6 | 1074 行 | 日结盈亏报告生成（含 `daily_reports` 表迁移） |
| 4 | [paper-to-live-switch](./2026-07-16-paper-to-live-switch.md) | 6 | 945 行 | Paper→Live 实盘切换安全机制（TradingMode + SafetyCheck） |
| 5 | [position-reconciliation](./2026-07-16-position-reconciliation.md) | 6 | 979 行 | 持仓同步与对账（含 `reconciliation_reports` 表迁移） |

### 用户体验（1 个）

| # | Plan | Tasks | 规模 | 说明 |
|---|------|-------|------|------|
| 7 | [first-run-wizard](./2026-07-16-first-run-wizard.md) | 5 | 865 行 | 首次启动向导（useFirstRun composable） |

### CI/CD 质量（3 个）

| # | Plan | Tasks | 规模 | 说明 |
|---|------|-------|------|------|
| 9 | [coverage-gate](./2026-07-16-coverage-gate.md) | 4 | 296 行 | 测试覆盖率门禁 |
| 10 | [goroutine-leak-ci](./2026-07-16-goroutine-leak-ci.md) | 5 | 262 行 | Goroutine 泄漏检测（go.uber.org/goleak） |
| 11 | [error-handling-audit](./2026-07-16-error-handling-audit.md) | 5 | 339 行 | 全项目错误处理审计（含扫描脚本） |

### 安全 / 配置（1 个）

| # | Plan | Tasks | 规模 | 说明 |
|---|------|-------|------|------|
| 12 | [api-key-management](./2026-07-16-api-key-management.md) | 5 | 645 行 | API Key 集中管理面板（apiKeyRegistry.ts） |

### 前端功能（2 个）

| # | Plan | Tasks | 规模 | 说明 |
|---|------|-------|------|------|
| 13 | [workflow-gallery](./2026-07-16-workflow-gallery.md) | 6 | 1012 行 | 6 套官方工作流模板库 |
| 14 | [user-manual](./2026-07-16-user-manual.md) | 7 | 1194 行 | 帮助中心 + 面板 ⓘ 内联帮助 |

### 性能（1 个）

| # | Plan | Tasks | 规模 | 说明 |
|---|------|-------|------|------|
| 15 | [startup-optimization](./2026-07-16-startup-optimization.md) | 5 | 914 行 | 启动优化（internal/startup/ 指标 + 优化器） |

---

## 依赖关系提示

- ~~release-pipeline~~ ✅ 已完成：auto-updater 需要 GitHub Releases 提供资产
- **error-visibility** 的 Toast 系统可被 paper-to-live-switch、position-reconciliation 等交易功能复用（错误提示）
- **coverage-gate / goroutine-leak-ci** 越早落地，后续 plan 的质量门槛越高
- 三个交易核心 plan 均涉及 SQLite 迁移，注意迁移序号不冲突

## 执行记录

进度 ledger: `.superpowers/sdd/progress.md`（git-ignored scratch，勿依赖长期保存）
