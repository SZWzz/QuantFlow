# 示例工作流库 (Workflow Gallery)

## Motivation

当前工作流引擎有 77 个节点，但用户打开 Workflow 模式时看到的是空白画布，不知道从哪里开始。虽然有节点调色板，但没有"完整工作流"的参考。

示例工作流解决：1) 演示引擎能力 2) 提供可运行的起点 3) 降低上手门槛。

## Design

### Gallery 面板

`frontend/src/terminal/panels/WorkflowGalleryPanel.vue` — 展示预置和用户保存的工作流：

```
┌──────────────────────────────────────────────┐
│  🖼 工作流库                                   │
├──────────────────────────────────────────────┤
│  🔍 [搜索...                               ] │
├──────────────────────────────────────────────┤
│  📂 官方示例 (6)                         全部导入│
│                                              │
│  ├ 🔀 金叉买入策略     ⭐⭐⭐ 1.2k runs    [导入]│
│  │   均线金叉信号 → 回测 → 绩效报告             │
│  │                                              │
│  ├ 📈 RSI 超卖反弹     ⭐⭐⭐ 892 runs     [导入]│
│  │   RSI < 30 信号 → 选股 → 回测               │
│  │                                              │
│  ├ 📊 多因子选股       ⭐⭐ 645 runs      [导入]│
│  │   4 个 Alpha 因子 → 打分 → 排名 → Top 10    │
│  │                                              │
│  ├ 🤖 AI 策略生成      ⭐⭐⭐ 1.5k runs    [导入]│
│  │   自然语言 → LLM → DAG → 回测 → 迭代        │
│  │                                              │
│  ├ 🔄 定时监控 + 通知  ⭐⭐ 312 runs      [导入]│
│  │   每分钟检查 RSI → 超卖 → Telegram 提醒      │
│  │                                              │
│  └ 🚀 动量突破系统     ⭐⭐⭐ 756 runs     [导入]│
│      扫描全市场 → 突破信号 → 模拟盘              │
│                                              │
│  ────────────────────────────────────────     │
│  📂 我的工作流 (3)                              │
│  ├ 我的策略 v2                     [编辑] [导出]│
│  ├ 日线回测 2026                  [编辑] [导出]│
│  └ AI 优化版                      [编辑] [导出]│
└──────────────────────────────────────────────┘
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/terminal/panels/WorkflowGalleryPanel.vue` | 新建 | Gallery 面板 |
| `frontend/src/workflow/gallery/official.ts` | 新建 | 6 个官方示例工作流定义 |
| `frontend/src/stores/workflow.ts` | 修改 | 新增 `importWorkflow`, `listUserWorkflows` |
| `internal/storage/workflow_repo.go` | 修改 | 支持保存/加载用户工作流 |
| `examples/` | 修改 | 增补示例 JSON 文件 |

### 6 个官方示例详细设计

每个示例包含：名称、描述、标签、节点数、JSON 定义

#### 1. 金叉买入策略

```
[DataLoader: 000001.SZ 1y] → [SMA: 5日] → [CrossSignal: 金叉]
  → [BacktestNode: A股规则] → [ChartOutput]
                ↓
          [PerformanceReport]
```

#### 2. RSI 超卖反弹

```
[DataLoader: 沪深300 6m] → [RSI: 14日] → [Threshold: <30 信号]
  → [StockScanner: A股全市场] → [BacktestNode] → [Output]
```

#### 3. 多因子选股

```
[DataLoader: 全A股 1y] → [Factor: momentum_1m] → [Factor: volatility_20d]
  → [Factor: volume_ratio] → [Factor: alpha_191]
  → [Score: 加权排名] → [Filter: Top10] → [Output]
```

#### 4. AI 策略生成

```
[UserInput: "写一个低波动率选股策略"] → [Agent: LLM 推理]
  → [DAG Generator] → [BacktestNode] → [Agent: 迭代优化]
  → [Output]
```

#### 5. 定时监控 + 通知

```
[Schedule: 每分钟] → [DataLoader: 自选股] → [RSI: 14日]
  → [Condition: RSI < 30] → [Notify: Telegram]
```

#### 6. 动量突破系统

```
[Schedule: 每日开盘] → [DataLoader: 全A股] → [SMA: 20日]
  → [Volume: 放量检测] → [CrossSignal: 突破]
  → [Filter: 市值>100亿] → [PlaceOrder: Paper模式]
```

### 示例工作流格式

```typescript
// frontend/src/workflow/gallery/official.ts
interface GalleryWorkflow {
  id: string
  name: string
  nameZh: string
  description: string
  descriptionZh: string
  tags: string[]
  difficulty: 'beginner' | 'intermediate' | 'advanced'
  nodes: number
  estimatedRuns: number
  json: WorkflowJSON  // 完整的 vue-flow 兼容工作流定义
}

export const OFFICIAL_WORKFLOWS: GalleryWorkflow[] = [
  {
    id: 'golden-cross',
    name: 'Golden Cross Strategy',
    nameZh: '金叉买入策略',
    description: '5-day SMA crosses above 20-day SMA → buy signal → backtest',
    descriptionZh: '5日均线上穿20日均线 → 买入信号 → A股回测验证',
    tags: ['均线', '回测', '入门'],
    difficulty: 'beginner',
    nodes: 8,
    estimatedRuns: 1200,
    json: { ... }  // 完整的 WorkflowJSON
  },
  // ... 5 more
]
```

### 导入流程

用户点击"导入" → `workflowStore.importWorkflow(workflow.json)` → 工作流出现在画布 → 用户可立即运行或编辑。

导入时自动检查节点版本兼容性（比如 WorkflowJSON 引用的节点类型在当前 engine 中是否注册）。

### 社区工作流（v2 规划）

- 用户可导出工作流为 JSON 分享
- (未来) 工作流市场 → 用户上传 / 下载

## Acceptance Criteria

- [ ] WorkflowGalleryPanel 展示官方示例（6个）和用户工作流
- [ ] 每个示例显示名称、描述、难度、节点数、import 次数（mock 数据）
- [ ] 搜索框按名称/描述/标签过滤
- [ ] 点击"导入" → 工作流出现在 WorkflowCanvas
- [ ] 导入时校验节点兼容性（不兼容时提示但不阻止）
- [ ] 用户工作流显示在"我的工作流"分区
- [ ] 用户工作流可编辑、导出、删除
- [ ] 用户工作流持久化到 SQLite
- [ ] 6 个示例工作流的 JSON 都经过手动验证（可在 canvas 中加载并运行）
- [ ] 前端测试覆盖 Gallery 面板渲染 + 导入流程

## Risks / Trade-offs

- **风险**: 示例工作流可能因 API 变化过期（如 DataLoader 节点参数重命名）。→ CI 测试定期加载并验证 JSON schema
- **风险**: 社区工作流引入恶意 JSON。→ v1 仅官方示例，v2 社区需 schema 校验 + 沙箱执行
- **Trade-off**: 不实现运行次数统计（需要后端 + 数据库），v1 用 mock 数据展示
