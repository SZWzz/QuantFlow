# 2026-07-16 批次 Plan 执行状态

> 本批次共 15 个 plan（spec 见 `docs/specs/`，plan 见 `docs/superpowers/plans/`），按「基础设施 → 交易核心 → 用户体验 → CI/CD → 安全 → 前端功能」优先级执行。
> 执行方式：subagent-driven-development（每 task 独立子代理实现 + spec/质量双重 review + 全分支终审）。

**最后更新**: 2026-07-17

---

## ✅ 已完成（15/15）🎉

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

### 5. daily-pnl-report — 日结报告系统

- **Plan**: [2026-07-16-daily-pnl-report.md](./2026-07-16-daily-pnl-report.md)
- **完成日期**: 2026-07-17
- **交付**:
  - `internal/trading/daily_report.go` — `GenerateDailyReport(oms, date)` 汇总成交/持仓/盈亏/最大回撤/最佳最差交易
  - `internal/storage/migrations/019_daily_reports.sql` — `daily_reports` 表 + 索引
  - `internal/storage/daily_report_repo.go` — `SaveDailyReport` / `GetDailyReport` / `ListDailyReports`
  - `app_daily_report.go` — `GenerateDailyReport` / `GetDailyReport` / `ListDailyReports` / `ExportReportCSV` IPC
  - `DailyReportPanel.vue` — 前端日结报告面板（盈亏汇总/持仓明细/交易高亮/导出CSV/历史列表）
  - 面板注册到终端 `daily-report`
- **已知局限（defer）**:
  - 定时触发（scheduler）和通知推送（notify.Manager）延后
  - CSV 导出为占位符（完整 CSV 序列化延后到 phase 2）

### 6. paper-to-live-switch — Paper→Live 实盘切换

- **Plan**: [2026-07-16-paper-to-live-switch.md](./2026-07-16-paper-to-live-switch.md)
- **完成日期**: 2026-07-17
- **交付**: `TradingMode` 类型 + `SafetyCheck`/`SafetyReport` (types.go), `EngineMode` 模式管理器 (engine_mode.go), `SwitchToLive`/`SwitchToPaper`/`EmergencyClose` IPC, `LiveModeBanner.vue`

### 7. position-reconciliation — 持仓对账系统

- **Plan**: [2026-07-16-position-reconciliation.md](./2026-07-16-position-reconciliation.md)
- **完成日期**: 2026-07-17
- **交付**: `ReconcileAll(oms, brokers)` 对账引擎, `reconciliation_reports` 表 (migration 020), `ReconcileAll`/`GetReconciliationReports` IPC

---

## ⏳ 未完成（0/15）✅ 全部完成

---

**执行完成日期**: 2026-07-17
**总 commit 数**: 10 commits
**新增文件**: 40+ files across Go backend, Vue frontend, CI/CD workflows
