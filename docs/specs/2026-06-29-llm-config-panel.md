# LLM 配置面板（改造 ModelRegistryPanel）

## Motivation
用户无法配置大模型 API（OpenAI/Anthropic/DeepSeek/Ollama），ModelRegistryPanel 原为 ML 模型注册面板但对用户无用。将其改造为 LLM Provider 配置面板，提供 API Key / Base URL 管理、模型浏览、连接测试功能。

## Design

### 数据流
```
SettingsPanel (旧)                ModelRegistryPanel (新: LLM 配置)
┌─────────────────┐              ┌──────────────────────────┐
│ API Keys:        │              │ Provider Cards:           │
│  fred / finnhub  │              │  OpenAI      [key][url]   │
│  iwencai         │              │  Anthropic   [key][url]   │
└─────────────────┘              │  DeepSeek    [key][url]   │
                                 │  Ollama      [url]        │
                                 │                           │
                                 │ [Fetch Models]            │
                                 │ ┌─ Model List ──────────┐ │
                                 │ │ ollama/llama3.1:8b    │ │
                                 │ │ openai/gpt-4o         │ │
                                 │ │ anthropic/claude-...  │ │
                                 │ └───────────────────────┘ │
                                 │ [Default Model Selector]   │
                                 │ [Test Connection]          │
                                 └──────────────────────────┘
                                        │ localStorage
                                        ▼
                                  settingsStore (前端)
                                        │ UpdateConfig
                                        ▼
                                  config.yaml (Go)
                                        │ env vars / gRPC
                                        ▼
                                  Python LLM Provider

```

### 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/terminal/panels/ModelRegistryPanel.vue` | 重写 | 改为 LLM 配置面板，保留基本布局 |
| `frontend/src/stores/settings.ts` | 新增字段 | `llmOpenaiKey`, `llmOpenaiBaseUrl`, `llmAnthropicKey`, `llmAnthropicBaseUrl`, `llmDeepseekKey`, `llmDeepseekBaseUrl`, `llmOllamaBaseUrl`, `llmDefaultModel` |
| `frontend/src/stores/ml.ts` | 新增 action | `fetchAvailableModels()`, `testLLMConnection()` |
| `frontend/src/lib/i18n/zh.ts` | 新增键 | `settings.ai`, `settings.llm_*` |
| `frontend/src/lib/i18n/en.ts` | 新增键 | 同上英文 |
| `frontend/src/lib/composables/usePanelCache.ts` | 新增 | 已有，复用 |
| `app/config.go` | 新增字段 | `Config.LLMApiKeys` |
| `app/app.go` | 新增方法 | `App.ListLLMModels()`, `App.TestLLMConnection()` |
| `python/proto/llm.proto` | 新增 rpc | `TestConnection` |
| `python/src/llm/engine.py` | 新增 handler | `TestConnection` |

### API 变更

**Go → 前端暴露:**
- `App.ListLLMModels() → LLMModelInfo[]` — 调用 Python ListModels gRPC
- `App.TestLLMConnection(provider, apiKey, baseUrl) → {success, error, latencyMs}` — 测试连接

**Python gRPC 新增:**
- `TestConnection(TestConnectionRequest) → TestConnectionResponse`

## 验收标准
- [ ] 面板展示 4 个 Provider 卡片（OpenAI/Anthropic/DeepSeek/Ollama）
- [ ] 每个卡片可编辑 API Key（密码框）和 Base URL
- [ ] 保存后写入 localStorage，重启 sidecar 后生效
- [ ] "Fetch Models" 按钮从 Python 拉取可用模型列表
- [ ] 模型列表可筛选/搜索
- [ ] 可设置默认模型，AIChatPanel 联动
- [ ] 全部 i18n 中英文
- [ ] full build 通过

## 风险/权衡
- **运行时生效**: Proto gRPC `TestConnection` 需要新增 proto 并重新生成（可做可不做，不做则 fallback 到前端模拟）
- **ModelRegistryPanel 原功能丢失**: 原面板没有用户实际在用 ML 模型管理，故直接覆盖
