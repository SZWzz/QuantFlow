# API Key 集中管理面板

## Motivation

QuantFlow 对接 40+ 数据源和 4 家券商，大部分需要 API Key / Secret / Token。当前 SettingsPanel 已有少量输入框（QOS），但：1) 覆盖不全 2) 无健康检测 3) 无集中概览。用户不知道哪些 key 已配、哪些缺失、哪些失效。

## Design

### 数据流

```
SettingsPanel API Keys 标签页
  → 列出所有已知 key 源 (常量列表)
  → 每个源展示: 名称 + 状态指示器 + 输入框 + 验证按钮
  → 输入/修改 → CredentialManager.Save() → 加密存储到 OS 密钥链
  → 验证 → CredentialManager.Verify() → 真实 API 调用测试
  → 状态缓存到 localStorage (避免每次启动都验证)
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/terminal/components/ApiKeyManager.vue` | 新建 | 独立 API Key 管理组件（内嵌 SettingsPanel） |
| `frontend/src/lib/apiKeyRegistry.ts` | 新建 | 所有已知 key 的注册表 |
| `internal/auth/credential.go` | 追加 | `Verify(service string) error` 方法 |
| `app_system.go` | 追加 | `VerifyApiKey(service string) error` IPC |

### API Key 注册表

```typescript
// frontend/src/lib/apiKeyRegistry.ts
export interface ApiKeyEntry {
  id: string
  name: string           // 显示名称
  market: string[]       // 关联市场
  type: 'api_key' | 'secret' | 'token' | 'both'
  required: boolean      // 核心功能是否必须
  verifyURL?: string     // 验证端点
  docURL?: string        // 如何获取 key 的文档
}

export const API_KEY_REGISTRY: ApiKeyEntry[] = [
  { id: 'tushare',       name: 'TuShare',       market: ['CN'],           type: 'api_key', required: false },
  { id: 'polygon',       name: 'Polygon.io',    market: ['US'],           type: 'api_key', required: false },
  { id: 'qos',           name: 'QOS',            market: ['HK'],           type: 'api_key', required: false },
  { id: 'futu_opend',    name: '富途 OpenD',     market: ['CN','HK','US'], type: 'token',   required: false },
  { id: 'alpaca_key',    name: 'Alpaca API Key', market: ['US'],           type: 'both',    required: false },
  { id: 'binance_key',   name: 'Binance API Key',market: ['CRYPTO'],       type: 'both',    required: false },
  { id: 'openai_key',    name: 'OpenAI',         market: ['ALL'],           type: 'api_key', required: false },
  { id: 'deepseek_key',  name: 'DeepSeek',       market: ['ALL'],           type: 'api_key', required: false },
  // ... 全部 15-20 个条目
]
```

### 状态指示器

```
┌──────────────────────────────────────┐
│  🔑 API Keys                         │
├──────────────────────────────────────┤
│  ✅ TuShare          [········] 🔍   │  ← 已配 + 已验证
│  ⚠️ Polygon.io       [········] 🔍   │  ← 已配但验证失败
│  ❌ QOS              [········] 🔍   │  ← 未配置
│  ➕ OpenAI           [········] 🔍   │  ← 可选未配置
│  ✅ Binance API      [Key·Sec] 🔍   │  ← both 类型展示双输入框
│  ...                                │
│                                      │
│  [全部验证]   [导出配置]  [导入配置]   │
└──────────────────────────────────────┘
```

### 验证机制

- `Verify(serviceID string) error`: 发起一次真实 API 调用（如 TuShare `token=` 测试）
- 验证结果缓存 5 分钟到 `localStorage`
- 面板底部"全部验证"按钮并行验证所有已配 key
- 验证失败不阻止使用，只标记 ⚠️

### 导入/导出

- 导出：加密打包所有 key 为 `quantflow-keys.json.enc`（AES-GCM）
- 导入：解密后批量写入 CredentialManager

## Acceptance Criteria

- [ ] SettingsPanel 新增 "API Keys" 标签页，列出 15-20 个数据源/券商 key 条目
- [ ] 每个条目展示状态指示器（✅/⚠️/❌/➖），颜色编码
- [ ] 输入 key 后通过 IPC → CredentialManager.Save() 加密存储
- [ ] "验证"按钮真实 API 调用并更新状态
- [ ] "全部验证"按钮并行执行
- [ ] 导出/导入加密 key 包功能
- [ ] 前端单元测试覆盖 ApiKeyManager.vue
- [ ] Go 侧测试覆盖 `Verify()` mock API

## Risks / Trade-offs

- **风险**: 验证调用可能触发 API 限流。→ 两次验证之间至少间隔 60s
- **风险**: 导出加密包的 key 管理责任。→ README 中说明安全注意事项
- **Trade-off**: 不在启动时自动验证全部 key（太慢），按需 + 手动触发
