# 自动更新 (Auto-Updater)

## Motivation

QuantFlow 是桌面应用，无自动更新意味着用户永远停留在下载时的版本。Bug 修复和新功能无法触达用户。Wails v3 不提供内置 updater，需要自行实现。

目标：用户无感更新，重启即生效。

## Design

### 更新策略：检查 → 下载 → 替换 → 重启

```
App 启动
  ↓
启动 30s 后 (延迟检查，不拖慢启动)
  ↓
GET github.com/SZWzz/QuantFlow/releases/latest
  → 获取最新版本号
  → 与本版本比较
  ↓
有新版本?
  ├─ 否 → 静默结束
  └─ 是 → 用户提示:
      ┌──────────────────────────────┐
      │  新版本 v2026.7.20 可用        │
      │  当前: v2026.7.14              │
      │  [稍后提醒]  [查看更新]  [更新] │
      └──────────────────────────────┘
       ↓
     用户确认 → 后台下载
       ↓
    下载完成 → 校验 checksum
       ↓
    校验通过 → 替换二进制 + 重启
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/updater/updater.go` | 新建 | 更新引擎 (检查+下载+校验+替换) |
| `internal/updater/release.go` | 新建 | GitHub Releases API 客户端 |
| `internal/updater/darwin.go` | 新建 | macOS 特定替换逻辑 (移动 .app bundle) |
| `internal/updater/linux.go` | 新建 | Linux 特定替换逻辑 |
| `internal/updater/windows.go` | 新建 | Windows 特定替换逻辑 |
| `app_system.go` | 追加 | `CheckUpdate() UpdateInfo`, `ApplyUpdate()` IPC |
| `frontend/src/terminal/components/UpdatePrompt.vue` | 新建 | 更新提示对话框 |

### 更新检查协议

```
GET /repos/SZWzz/QuantFlow/releases/latest
  ↓
Response:
{
  "tag_name": "v2026.7.20",
  "assets": [
    {
      "name": "quantflow-darwin-amd64.zip",
      "browser_download_url": "...",
      "size": 52428800
    },
    {
      "name": "quantflow-darwin-arm64.zip",
      "browser_download_url": "..."
    },
    {
      "name": "quantflow-linux-amd64.tar.gz",
      "browser_download_url": "..."
    },
    {
      "name": "quantflow-windows-amd64.zip",
      "browser_download_url": "..."
    }
  ],
  "body": "Bug fixes: ..."
}
```

### 校验机制

- Release 发布时生成 SHA256 checksum 文件（如 `quantflow-darwin-arm64.zip.sha256`）
- 下载完成后计算本地 SHA256，匹配才执行替换
- 匹配失败：删除下载文件，提示用户手动下载

### 替换策略

| 平台 | 策略 |
|------|------|
| macOS | 下载 `.app` bundle → 替换 `/Applications/QuantFlow.app` → `open -a QuantFlow` |
| Linux | 下载 tar.gz → 解压替换二进制 → exec 新进程 |
| Windows | 下载 zip → 替换 `.exe` → 启动新进程 |

所有平台：新二进制放在临时目录 → 当前进程启动新二进制 → 当前进程退出。

### 数据流

```
前端: updateStore.check()
  → IPC CheckUpdate()
    → updater.Check() → HTTP GET GitHub API
    → 返回 UpdateInfo{HasUpdate, LatestVersion, ReleaseURL, AssetURL, Size, Checksum}
  → 有更新 → UpdatePrompt 组件弹出
  → 用户确认 → IPC ApplyUpdate(assetURL, checksum)
    → updater.Download() → updater.Verify() → updater.Replace() → restart
```

### 更新频率

- 启动 30s 后检查
- Settings 可配：每次启动 / 每天一次 / 从不
- 手动：Help → Check for Updates

## Acceptance Criteria

- [ ] 启动后延迟检查最新版本，不拖慢启动
- [ ] GitHub Releases API 正确解析最新 release 和平台对应 asset
- [ ] 有新版本时弹出 UpdatePrompt（带 changelog 摘要）
- [ ] 下载过程显示进度条（通过 ws 推送下载百分比）
- [ ] SHA256 校验失败不替换，提示用户手动下载
- [ ] 替换后自动重启应用
- [ ] Settings 中有更新频率选项（每次/每天/从不）
- [ ] "Help → Check for Updates" 手动触发
- [ ] Go 测试覆盖版本比较、asset 匹配、checksum 校验
- [ ] 平台特定替换逻辑 mock 测试

## Risks / Trade-offs

- **风险**: GitHub API 在中国大陆可能不可达。→ 添加 Gitee/Gitea mirror 配置
- **风险**: 替换二进制可能被杀软拦截（尤其 Windows）。→ 代码签名可缓解
- **风险**: 下载中断导致损坏。→ resume 支持 + checksum 校验
- **Trade-off**: 不实现 delta 更新（只下载完整二进制）——实现简单，二进制 ~50MB 可接受
- **Trade-off**: 不自建更新服务器，纯 GitHub Releases，零运维成本
