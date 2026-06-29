# 实施计划: LLM 配置面板

## 任务拆分

### 任务 1: Settings store 新增 LLM 字段
文件: `frontend/src/stores/settings.ts`
- 新增 `llmOpenaiKey`, `llmOpenaiBaseUrl`, `llmAnthropicKey`, `llmAnthropicBaseUrl`, `llmDeepseekKey`, `llmDeepseekBaseUrl`, `llmOllamaBaseUrl`, `llmDefaultModel` 到 SettingsState
- 添加到 defaultSettings()
- 全部初始化为空字符串

### 任务 2: i18n 新增 LLM 配置相关键
文件: `frontend/src/lib/i18n/zh.ts`, `frontend/src/lib/i18n/en.ts`
- `settings.ai: 'AI / 大模型' / 'AI / LLM'`
- `settings.llm_providers: '模型供应商'`
- `settings.llm_openai_key / settings.llm_anthropic_key / settings.llm_deepseek_key: 'API Key'`
- `settings.llm_openai_url / settings.llm_anthropic_url / settings.llm_deepseek_url / settings.llm_ollama_url: '接口地址'`
- `settings.llm_default_model: '默认模型'`
- `settings.llm_fetch_models: '获取模型列表'`
- `settings.llm_test: '测试连接'`
- `settings.llm_test_success / settings.llm_test_fail: '连接成功' / '连接失败'`
- `settings.llm_models_loaded: 已加载 {count} 个模型`
- `settings.llm_no_models: 暂无模型数据 — 请先配置 API Key 后点击获取`
- `settings.llm_save_hint: '保存后将在下次重启 sidecar 时生效'`

### 任务 3: 重写 ModelRegistryPanel.vue
文件: `frontend/src/terminal/panels/ModelRegistryPanel.vue`
- Provider 卡片网格: 4 个 provider 各一个卡片
- 每个卡片: provider 名称 + 图标 emoji + API Key 输入(密码类型) + Base URL 输入 + Test Connection 按钮
- 底部: Fetch Models 按钮 + 模型列表表格 + 默认模型下拉
- 保存按钮: 写入 settingsStore
- 加载时从 settingsStore 读取已有值

### 任务 4: Go 端新增 ListLLMModels
文件: `app/app.go` 或新建 `app/app_llm.go`
- `func (a *App) ListLLMModels() ([]map[string]interface{}, error)`
- 调用 `a.bridge.ListModels(ctx)` 并转换 protobuf 结果为 map
- 前端通过 `window.go.main.App.ListLLMModels()` 调用

### 任务 5: 集成 — AIChatPanel 使用默认模型
文件: `frontend/src/terminal/panels/AIChatPanel.vue`
- `selectedModel` 初始值从 settingsStore 读取 `llmDefaultModel`
- `availableModels` 默认从 settingsStore 读取（或保持硬编码，等将来 Python 连接后动态获取）

### 任务 6: 验证 build
- `vue-tsc --noEmit` 通过
- `wails3 build` 通过
