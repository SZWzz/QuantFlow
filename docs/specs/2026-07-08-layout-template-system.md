# 布局模板系统 (Layout Template System)

## Motivation

当前 QuantFlow 终端只有一个匿名布局，保存在 `localStorage`，无法命名保存/恢复多个布局配置。用户需要为不同场景（交易、研究、盘前简报）切换不同面板排布。

## Design

### 数据流

```
用户保存布局 → terminalStore.saveLayout("trading")
                   ↓
              localStorage (快速回退)
              + Wails IPC → Go App.SaveLayout()
                   ↓
              SQLite user_config 表 (key=layout:<name>, value=JSON)

用户加载布局 → terminalStore.loadLayout("trading")
                   ↑
              localStorage (缓存) or IPC → Go App.LoadLayout()
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/internal/storage/migrations/018_user_config.sql` | 新建 | user_config 表 (key-value) |
| `app/internal/storage/migrate.go` | 修改 | 注册 migration 018 |
| `app/app_data.go` | 追加 | 3 个 IPC 方法 (Save/Load/List/Delete Layout) |
| `app/app_data_test.go` | 追加 | IPC 集成测试 |
| `frontend/src/lib/wails.ts` | 修改 | 添加 layout IPC 类型绑定 |
| `frontend/src/stores/terminal.ts` | 修改 | 添加 `savedLayouts`, `saveLayout()`, `loadLayout()`, `deleteLayout()` |
| `frontend/src/terminal/panels/LayoutTemplatePanel.vue` | 新建 | 布局模板管理面板 |
| `frontend/src/terminal/panels/registry.ts` | 修改 | 注册 LayoutTemplatePanel |
| `frontend/src/lib/i18n/zh.ts` | 修改 | 中文 i18n |
| `frontend/src/lib/i18n/en.ts` | 修改 | 英文 i18n |
| `frontend/src/terminal/DockView/DockView.vue` | 修改 | 添加 Ctrl+Shift+数字 快捷键载入布局 |
| `CHANGELOG.md` | 修改 | 记录变更 |

### API 变更

#### Go IPC 方法 (app_data.go 追加)

```go
// SaveLayout stores the current layout JSON under a named key.
func (a *App) SaveLayout(ctx context.Context, name string, layoutJSON string) error

// LoadLayout retrieves a previously saved layout JSON by name.
func (a *App) LoadLayout(ctx context.Context, name string) (string, error)

// ListLayouts returns all saved layout names.
func (a *App) ListLayouts(ctx context.Context) ([]string, error)

// DeleteLayout removes a saved layout.
func (a *App) DeleteLayout(ctx context.Context, name string) error
```

#### SQLite Schema

```sql
-- 018_user_config: key-value config store for UI state
CREATE TABLE IF NOT EXISTS user_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

Layouts stored with key prefix `layout:<name>`, value = full `DockLayoutTree` JSON.

#### Pinia Store 变更 (terminalStore)

```typescript
// New state
savedLayouts: ref<string[]>([])

// New actions
saveLayout(name: string): Promise<void>
loadLayout(name: string): Promise<void>
deleteLayout(name: string): Promise<void>
refreshLayouts(): Promise<void>
```

#### Keyboard Shortcuts

| 快捷键 | 当前 | 新增 |
|--------|------|------|
| Ctrl+1 | 预设: 单面板 | 保持不变 |
| Ctrl+2 | 预设: 水平分割 | 保持不变 |
| Ctrl+3 | 预设: 2x2 网格 | 保持不变 |
| Ctrl+4 | 预设: 侧边栏 | 保持不变 |
| Ctrl+Shift+1..9 | — | 载入已保存布局 #1..#9 |

## Acceptance Criteria

- [ ] 018 迁移创建 `user_config` 表
- [ ] `SaveLayout` / `LoadLayout` / `ListLayouts` / `DeleteLayout` IPC 方法实现 + 测试
- [ ] 前端 `terminalStore` 可以保存/加载/删除布局
- [ ] 布局保存到 SQLite；同时缓存到 localStorage 实现离线回退
- [ ] LayoutTemplatePanel 展示已保存布局列表，支持保存当前布局、载入、删除
- [ ] Ctrl+Shift+1..9 快捷键载入对应布局
- [ ] 中文/英文 i18n 文本
- [ ] `go vet` + `go test` 通过
- [ ] `npx vue-tsc --noEmit` + `npx vitest run` 通过

## Risks / Trade-offs

- **localStorage 优先**: 读取布局时先检查 localStorage 缓存，再回退到 IPC，减少 Wails bridge 延迟
- **无版本兼容**: 布局 JSON 格式随前端演进可能变化，旧布局载入时可能出现不兼容。采用"静默降级"——无法解析的 tab 跳过，保留可解析部分
- **无认证**: 单用户桌面应用，无需用户级隔离
