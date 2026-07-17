# 用户手册 (User Manual)

## Motivation

QuantFlow 有 74 个面板、77 个工作流节点、40+ 数据源、4 家券商、Python sidecar、AI Agent……但没有任何一份用户能看的文档。唯一的存在是 README 的粗略概览和 CLAUDE.md（给 AI 看的开发规范）。

用户手册解决：1) 新用户不知道终端能做什么 2) 老用户不知道某个功能在哪 3) 配置数据源/券商时需要 step-by-step 指引。

## Design

### 手册结构

用户手册不写在代码仓库里（用户不会去看 markdown），而是**内嵌到终端中**，通过以下两种方式交付：

#### 方式 A: 内置帮助终端 (Help Panel)

新增 `HelpPanel`，内容从 JSON 数据源加载（构建时嵌入）：

```
┌──────────────────────────────────────────────┐
│  📖 QuantFlow 帮助中心                        │
├──────────────────────────────────────────────┤
│  🔍 [搜索...                               ] │
├──────────────────────────────────────────────┤
│  📚 快速入门                                  │
│    ├ 首次启动向导                              │
│    ├ 添加自选股                                │
│    └ 运行第一个回测                            │
│                                              │
│  🖥 面板参考                                   │
│    ├ 行情 (17) →                              │
│    ├ 交易 (8)  →                              │
│    ├ 研究 (10) →                              │
│    └ 系统 (7)  →                              │
│                                              │
│  🔌 数据源配置                                  │
│    ├ 配置 A 股数据源                            │
│    ├ 配置美股数据源                              │
│    └ 检查数据源状态                             │
│                                              │
│  🤖 工作流                                     │
│    ├ 创建第一个工作流                            │
│    ├ 节点参考 (77) →                           │
│    └ 示例工作流                                 │
│                                              │
│  🏦 券商接入                                    │
│    ├ Alpaca 配置                                │
│    ├ Binance 配置                               │
│    ├ IBKR 配置                                  │
│    └ 富途配置                                    │
│                                              │
│  🐍 Python 高级版                               │
│    ├ 安装 Python sidecar                       │
│    ├ 自定义因子                                  │
│    └ LLM 配置                                    │
│                                              │
│  🛠 设置与维护                                   │
│    ├ 主题切换                                    │
│    ├ 布局管理                                    │
│    ├ 快捷键参考                                   │
│    └ 数据备份与恢复                               │
└──────────────────────────────────────────────┘
```

#### 方式 B: 面板内嵌引导 (Tooltip + Hint)

每个面板右上角 `ⓘ` 按钮 → 弹出面板用途说明、数据来源、关键操作：

```
┌──────────────────────┐
│  WatchlistPanel       │
│                       │
│  ⓘ 自选股列表         │
│  展示用户关注的股票    │
│  实时价格 + 涨跌幅     │
│                       │
│  数据源: Tencent/Sina │
│  快捷键: Ctrl+W 搜索  │
│  右键菜单添加/删除    │
└──────────────────────┘
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/terminal/panels/HelpPanel.vue` | 新建 | 帮助中心面板 |
| `frontend/docs/manual/quickstart.json` | 新建 | 快速入门（结构化 JSON） |
| `frontend/docs/manual/panels.json` | 新建 | 面板参考（74 个面板说明） |
| `frontend/docs/manual/adapters.json` | 新建 | 数据源配置指南 |
| `frontend/docs/manual/nodes.json` | 新建 | 节点参考 |
| `frontend/docs/manual/brokers.json` | 新建 | 券商配置指南 |
| `frontend/docs/manual/python.json` | 新建 | Python sidecar 指南 |
| `frontend/docs/manual/settings.json` | 新建 | 设置与维护指南 |
| `frontend/src/lib/usePanelHelp.ts` | 新建 | 面板 help 内容提供 |
| `frontend/src/terminal/components/PanelHelpPopover.vue` | 新建 | 面板 ⓘ 弹出组件 |
| `frontend/src/terminal/panels/registry.ts` | 修改 | 注册 HelpPanel (ID: `HelpPanel`) |

### JSON 内容格式

```json
// frontend/docs/manual/panels.json
{
  "panels": [
    {
      "id": "WatchlistPanel",
      "name": "自选股列表",
      "category": "行情",
      "description": "展示用户关注的股票实时价格和涨跌幅。支持多分组、排序、右键菜单。",
      "dataSources": ["Tencent", "Sina"],
      "shortcut": "Ctrl+W 添加股票",
      "tips": [
        "双击股票打开 CandlestickPanel",
        "右键 → Add to Workflow 生成行情节点",
        "分组名可拖拽排序"
      ],
      "relatedPanels": ["CandlestickPanel", "MarketOverviewPanel"],
      "configurable": true
    }
  ]
}
```

### 构建时嵌入

手册 JSON 放在 `frontend/docs/manual/`，Vite 构建时自动打包进 JS bundle，不增加运行时请求。

或：JSON 转为 TypeScript 常量（`manual.ts`），Tree-shaking 友好的 import 方式。

### 更新流程

- 新增面板/节点时，作者同步更新对应 JSON 文件
- 可在 CHANGELOG entry 中提及"文档已更新"
- 定期检查 JSON 中的面板 ID 与 registry.ts 是否一致（通过 CI 测试）

## Acceptance Criteria

- [ ] HelpPanel 可添加到 DockView（ID: HelpPanel）
- [ ] 帮助中心按类别组织（快速入门、面板参考、数据源、工作流、券商、Python、设置）
- [ ] 搜索功能模糊匹配所有内容
- [ ] 每个面板的 ⓘ 弹出显示用途、数据源、快捷键、常见操作
- [ ] 面板参考覆盖全部 74 个面板（ID + 名称 + 描述 + 截图文字说明）
- [ ] 数据源配置指南覆盖全部 40+ 适配器
- [ ] 工作流节点参考覆盖全部 77 节点
- [ ] 券商配置指南覆盖 Alpaca/Binance/IBKR/富途
- [ ] JSON 文件经 schema 校验测试（确保 ID 与 registry 一致）
- [ ] CI 测试验证 panel JSON 与 registry 无 drift

## Risks / Trade-offs

- **风险**: 文档跟不上代码变更。→ JSON 与 registry 的 CI 一致性检查 + CHANGELOG 文档提醒
- **风险**: 用户不看帮助文档。→ 面板 ⓘ 的"及时帮助"覆盖率更重要
- **Trade-off**: 不做外部 wiki 或网站（维护负担），全部内嵌到终端中
